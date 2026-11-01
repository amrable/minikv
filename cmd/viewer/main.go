package main

import (
	"context"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/amrable/minikv"
	"github.com/amrable/minikv/internal/skiplist"
	"github.com/amrable/minikv/internal/sstmanager"
	"github.com/amrable/minikv/internal/wal"
)

//go:embed index.html
var page embed.FS

type viewer struct {
	db  *minikv.KV
	sst *sstmanager.SSTManager
	wal *wal.WAL
	dir string
	mu  sync.Mutex
}

type item struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Hex     string `json:"hex"`
	Deleted bool   `json:"deleted"`
}

type logItem struct {
	Operation string `json:"operation"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Hex       string `json:"hex"`
}

type tableView struct {
	Name    string `json:"name"`
	Bytes   int    `json:"bytes"`
	Entries []item `json:"entries"`
}

type state struct {
	Directory     string      `json:"directory"`
	Memtable      []item      `json:"memtable"`
	MemtableBytes int         `json:"memtableBytes"`
	WALBytes      int64       `json:"walBytes"`
	WAL           []logItem   `json:"wal"`
	Tables        []tableView `json:"tables"`
	Visible       []item      `json:"visible"`
}

func main() {
	dir := flag.String("dir", "./minikv-data", "database directory")
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()

	if err := run(*dir, *addr); err != nil {
		log.Fatal(err)
	}
}

func run(dir, addr string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	logReader, err := wal.Open(dir)
	if err != nil {
		return fmt.Errorf("open WAL inspector: %w", err)
	}
	defer logReader.Close()
	db, err := minikv.Open(dir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	app := &viewer{
		db:  db,
		sst: sstmanager.New(dir),
		wal: logReader,
		dir: dir,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", app.index)
	mux.HandleFunc("GET /api/state", app.getState)
	mux.HandleFunc("GET /api/get", app.get)
	mux.HandleFunc("POST /api/put", app.put)
	mux.HandleFunc("POST /api/delete", app.delete)
	mux.HandleFunc("GET /api/scan", app.scan)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("MiniKV viewer at http://%s (database: %s)", addr, dir)
	serveErr := server.ListenAndServe()
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		_ = db.Close()
		return serveErr
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}

func (v *viewer) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := page.ReadFile("index.html")
	if err != nil {
		http.Error(w, "viewer page unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (v *viewer) getState(w http.ResponseWriter, r *http.Request) {
	v.mu.Lock()
	defer v.mu.Unlock()
	snapshot, err := v.snapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (v *viewer) get(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	v.mu.Lock()
	value, found, err := v.db.Get(key)
	v.mu.Unlock()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key": key, "found": found, "value": string(value), "hex": hex.EncodeToString(value),
	})
}

func (v *viewer) put(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	v.mu.Lock()
	err := v.db.Put(input.Key, []byte(input.Value))
	v.mu.Unlock()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stored"})
}

func (v *viewer) delete(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Key string `json:"key"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	v.mu.Lock()
	err := v.db.Delete(input.Key, nil)
	v.mu.Unlock()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (v *viewer) scan(w http.ResponseWriter, r *http.Request) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	v.mu.Lock()
	snapshot, err := v.snapshot()
	v.mu.Unlock()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	result := make([]item, 0)
	for _, entry := range snapshot.Visible {
		if entry.Deleted || (from != "" && entry.Key < from) || (to != "" && entry.Key > to) {
			continue
		}
		result = append(result, entry)
	}
	writeJSON(w, http.StatusOK, result)
}

func (v *viewer) snapshot() (state, error) {
	dbState := v.db.DebugState()
	out := state{
		Directory:     v.dir,
		MemtableBytes: dbState.MemtableBytes,
		Memtable:      make([]item, 0, len(dbState.Memtable)),
		WAL:           make([]logItem, 0),
		Tables:        make([]tableView, 0),
	}
	visible := make(map[string]item)
	for _, entry := range dbState.Memtable {
		value := makeItem(entry.Key, entry.Value, entry.Deleted)
		out.Memtable = append(out.Memtable, value)
		visible[entry.Key] = value
	}

	if info, err := os.Stat(v.dir + string(os.PathSeparator) + "wal.log"); err == nil {
		out.WALBytes = info.Size()
	} else if !errors.Is(err, os.ErrNotExist) {
		return state{}, err
	}
	if err := v.wal.Replay(func(record wal.Record) error {
		logEntry := logItem{
			Operation: string(record.Type),
			Key:       record.Key,
			Value:     string(record.Value),
			Hex:       hex.EncodeToString(record.Value),
		}
		out.WAL = append(out.WAL, logEntry)
		return nil
	}); err != nil {
		return state{}, fmt.Errorf("read WAL: %w", err)
	}

	tableIndex := 0
	for data, err := range v.sst.Tables() {
		if err != nil {
			return state{}, err
		}
		tableIndex++
		table := tableView{Name: fmt.Sprintf("SST %d", tableIndex), Bytes: len(data), Entries: make([]item, 0)}
		for len(data) > 0 {
			tuple, consumed, err := skiplist.DecodeTuple(data)
			if err != nil {
				return state{}, fmt.Errorf("decode SST %d: %w", tableIndex, err)
			}
			entry := makeItem(tuple.Key, tuple.Value, tuple.Deleted)
			table.Entries = append(table.Entries, entry)
			if _, alreadySet := visible[tuple.Key]; !alreadySet {
				visible[tuple.Key] = entry
			}
			data = data[consumed:]
		}
		out.Tables = append(out.Tables, table)
	}

	keys := make([]string, 0, len(visible))
	for key := range visible {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out.Visible = make([]item, 0, len(keys))
	for _, key := range keys {
		out.Visible = append(out.Visible, visible[key])
	}
	return out, nil
}

func makeItem(key string, value []byte, deleted bool) item {
	return item{Key: key, Value: string(value), Hex: hex.EncodeToString(value), Deleted: deleted}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain one JSON value")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
