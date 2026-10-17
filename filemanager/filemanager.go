package filemanager

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func Write(dirName string, b []byte) error {
	path, err := nextName(dirName)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	dir, err := os.Open(dirName)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return err
	}
	return nil
}

func Walk(dirPath string) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, entry := range slices.Backward(entries) {
			if entry.IsDir() ||
				!strings.HasSuffix(entry.Name(), ".sst") {
				continue
			}

			path := filepath.Join(dirPath, entry.Name())
			payload, err := os.ReadFile(path)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(payload, nil) {
				return
			}
		}
	}
}

func nextName(dirPath string) (string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", err
	}
	for _, entry := range slices.Backward(entries) {
		if entry.IsDir() ||
			!strings.HasSuffix(entry.Name(), ".sst") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".sst")
		n, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		return fmt.Sprintf("%s/%010d.sst", dirPath, n+1), nil
	}
	return fmt.Sprintf("%s/%010d.sst", dirPath, 1), nil
}
