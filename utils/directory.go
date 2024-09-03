// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// GetFreeSpace returns the amount of free space in bytes on the filesystem containing the given path.
func GetFreeSpace(path string) (int64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return 0, err
	}
	return int64(fs.Bavail * uint64(fs.Bsize)), nil
}

// GetDirectorySize iterates over all files inside given directory (including subdirectories) and returns size in bytes.
func GetDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// Carmen can have files which are present at the call of this func but are not present when walk happens.
			// Hence, we ignore this error and file and continue with the rest of the files.
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
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
func GetDirectoryFiles(suffix string, paths []string) ([]string, error) {
	var files []string

	for _, path := range paths {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Check if the path represents a regular file (not a directory)
			if !info.IsDir() {
				if strings.HasSuffix(path, suffix) {
					files = append(files, path)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}
