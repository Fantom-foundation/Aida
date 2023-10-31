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
