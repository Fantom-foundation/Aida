package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// GetDirectorySize iterates over all files inside given directory (including subdirectories) and returns size in bytes.
func GetDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// GetDirectoryFiles returns all filenames within given directory.
// Note: Files inside any subdirectories are included.
func GetDirectoryFiles(path string) ([]string, error) {
	var files []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if the path represents a regular file (not a directory)
		if !info.IsDir() {
			if strings.HasSuffix(path, ".json") {
				files = append(files, path)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
