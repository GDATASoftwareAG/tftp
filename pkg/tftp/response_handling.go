package tftp

import (
	"io"
	"os"
)

type Handler interface {
	Matches(file string) bool
	Reader(file string) (io.ReadCloser, int64, error)
}

type ResponseHandling interface {
	RegisterHandler(handler Handler)
	OpenFile(file string) (io.ReadCloser, int64, error)
}

type responseHandling struct {
	handlers []Handler
}

func NewResponseHandling() ResponseHandling {
	return &responseHandling{}
}

func (f *responseHandling) RegisterHandler(handler Handler) {
	f.handlers = append(f.handlers, handler)
}

func (f *responseHandling) OpenFile(file string) (io.ReadCloser, int64, error) {
	var handler Handler
	for idx := range f.handlers {
		if f.handlers[idx].Matches(file) {
			handler = f.handlers[idx]
			break
		}
	}

	if handler == nil {
		return nil, 0, FileErrors.ErrNoHandler
	}

	reader, size, err := handler.Reader(file)
	if err != nil {
		return nil, 0, checkError(err)
	}
	return reader, size, nil
}

func checkError(err error) error {
	if os.IsPermission(err) {
		return FileErrors.ErrPermission
	} else if os.IsNotExist(err) {
		return FileErrors.ErrNotFound
	} else {
		return err
	}
}
