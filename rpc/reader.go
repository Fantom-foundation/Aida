// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/gzip"
)

//go:generate mockgen -source reader.go -destination reader_mocks.go -package rpc

type Iterator interface {
	Next() bool
	Value() *RequestAndResults
	Close()
	Error() error
}

// FileReader implements reader of the stored API recording
type FileReader struct {
	f *os.File
	Iterator
}

// NewFileReader creates new instance of the file reader and starts reading.
func NewFileReader(ctx context.Context, path string) (Iterator, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0640)
	if err != nil {
		return nil, err
	}

	var in io.ReadCloser

	// gzipped file?
	if strings.EqualFold(filepath.Ext(path), ".gz") {
		in, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
	} else {
		in = f
	}

	return &FileReader{
		f:        f,
		Iterator: newIterator(ctx, in, 10),
	}, nil
}

// Close the file reader releasing all the resources below.
func (fr *FileReader) Close() {
	fr.Iterator.Close()
	_ = fr.f.Close()
}
