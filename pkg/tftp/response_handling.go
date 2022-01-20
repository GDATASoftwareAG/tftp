package tftp

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gdatasoftwareag/tftp/v2/pkg/logging"
	"go.uber.org/zap"
)

type Handler interface {
	Matches(file string) bool
	Reader(ctx context.Context, file string) (io.ReadCloser, int64, error)
}

type ResponseHandling interface {
	RegisterHandler(handler Handler)
	OpenFile(ctx context.Context, file string) (io.ReadCloser, int64, error)
}

type responseHandling struct {
	handlers []Handler
	logger   logging.Logger
}

func NewResponseHandling(logger logging.Logger) ResponseHandling {
	return &responseHandling{
		logger: logger,
	}
}

func (f *responseHandling) RegisterHandler(handler Handler) {
	f.handlers = append(f.handlers, handler)
}

func (f *responseHandling) OpenFile(ctx context.Context, file string) (io.ReadCloser, int64, error) {
	var handler Handler
	for idx := range f.handlers {
		if f.handlers[idx].Matches(file) {
			f.logger.Debug("Found matching handler", zap.String("handler", fmt.Sprintf("%T", f.handlers[idx])))
			handler = f.handlers[idx]
			break
		}
	}

	if handler == nil {
		return nil, 0, FileErrors.ErrNoHandler
	}

	reader, size, err := handler.Reader(ctx, file)
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
