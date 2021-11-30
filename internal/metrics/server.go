package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Server interface {
	Start() <-chan error
	Shutdown(timeout time.Duration) error
}

type metricsServer struct {
	httpServer *http.Server
	errChan    chan error
}

func NewServerFromConfig(cfg Config, mux *http.ServeMux) Server {
	return &metricsServer{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: mux,
		},
	}
}

func (m *metricsServer) Start() <-chan error {
	m.errChan = make(chan error)
	go func() {
		m.errChan <- m.httpServer.ListenAndServe()
	}()
	return m.errChan
}

func (m *metricsServer) Shutdown(timeout time.Duration) error {
	if m.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		return m.httpServer.Shutdown(ctx)
	}

	close(m.errChan)
	return nil
}
