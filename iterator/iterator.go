package iterator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Iterator implements asynchronous record provider over a recorder API call set.
type Iterator struct {
	ctx    context.Context
	in     io.ReadCloser
	closed chan interface{}
	out    chan *RequestWithResponse
	item   *RequestWithResponse
	wg     *sync.WaitGroup
	err    error
}

// NewIterator creates a new API call records iterator over the data provided by an input reader.
func NewIterator(ctx context.Context, rc io.ReadCloser, queueLength int) *Iterator {
	i := Iterator{
		ctx:    ctx,
		in:     rc,
		closed: make(chan interface{}),
		out:    make(chan *RequestWithResponse, queueLength),
		wg:     new(sync.WaitGroup),
	}

	i.wg.Add(1)
	go i.load()

	return &i
}

// Next moves the iterator to the next item in the recording, if available.
// Returns FALSE if the iterator is exhausted.
func (i *Iterator) Next() bool {
	select {
	case <-i.ctx.Done():
		i.err = i.ctx.Err()
	case <-i.closed:
	case itm, open := <-i.out:
		if open {
			i.item = itm
			return true
		}
	}
	i.item = nil
	return false
}

// Close the iterator and release internal resources.
// The function waits for the internal routines to terminate.
func (i *Iterator) Close() {
	i.Release()
	i.wg.Wait()
}

// Release the internal resources of the iterator without raising an error.
func (i *Iterator) Release() {
	select {
	case <-i.closed:
	default:
		close(i.closed)
		_ = i.in.Close()
	}
}

// Value returns the value of the current element of the iterator.
// The value is empty when the iterator did not start by calling Next() first,
// or if the end of the content was reached.
func (i *Iterator) Value() *RequestWithResponse {
	return i.item
}

// Error returns an accumulated error of the iterator.
// Exhausting all available values via an iteration loop is not considered to be an error.
func (i *Iterator) Error() error {
	return i.err
}

// load the records from the given reader into the internal queue asynchronously.
func (i *Iterator) load() {
	defer i.wg.Done()
	defer close(i.out)

	for {
		req, err := i.read()
		if err != nil {
			// end of file is not propagated up; we just end the loading loop
			if err != io.EOF {
				i.err = err
			}
			return
		}

		select {
		case <-i.ctx.Done():
			i.err = i.ctx.Err()
			return
		case <-i.closed:
			return
		case i.out <- req:
		}
	}
}

// read next item from the reader for precessing.
func (i *Iterator) read() (*RequestWithResponse, error) {
	// try to read the next header
	hdr := new(Header)

	_, err := hdr.ReadFrom(i.in)
	if err != nil {
		return nil, err
	}

	method, err := hdr.Method()
	if err != nil {
		return nil, err
	}

	namespace, err := hdr.Namespace()
	if err != nil {
		return nil, err
	}

	return i.decode(hdr, namespace, method)
}

// decode loaded header into target structure.
func (i *Iterator) decode(hdr *Header, namespace, method string) (*RequestWithResponse, error) {
	// prep to read the payload
	req := RequestWithResponse{
		Query: &Body{
			Namespace:  namespace,
			MethodBase: method,
			Method:     fmt.Sprintf("%s_%s", namespace, method),
		},
		ParamsRaw:   make([]byte, hdr.QueryLength()),
		ResponseRaw: make([]byte, hdr.ResponseLength()),
	}

	err := i.loadPayload(req.ParamsRaw)
	if err != nil {
		return nil, err
	}

	// unmarshal parameters of the call
	err = json.Unmarshal(req.ParamsRaw, &req.Query.Params)
	if err != nil {
		return nil, err
	}

	if hdr.ResponseLength() > 0 {
		err = i.loadPayload(req.ResponseRaw)
		if err != nil {
			return nil, err
		}

		req.Response = &Response{
			BlockID:   hdr.BlockID(),
			Result:    req.ResponseRaw,
			Timestamp: hdr.BlockTimestamp(),
		}
	}

	// error?
	if hdr.IsError() {
		req.Error = &ErrorResponse{
			BlockID:   hdr.BlockID(),
			Timestamp: hdr.BlockTimestamp(),
			Error: ErrorMessage{
				Code: hdr.ErrorCode(),
			},
		}
	}

	return &req, nil
}

// loadPayload fills the given buffer with the payload loaded from the input reader
func (i *Iterator) loadPayload(buf []byte) error {
	var offset, n int
	var err error

	l := len(buf)
	for offset < l {
		n, err = i.in.Read(buf[offset:])
		offset += n

		if err != nil {
			break
		}
	}

	// we ignore any reported io error if we got the full package
	if offset == l {
		return nil
	}
	return err
}
