package metrics

import (
	"errors"
	"net/http"

	"github.com/gdatasoftwareag/tftp/internal/logging"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(logger logging.Logger, cfg Config, options ...ServerOption) Server {
	mux := http.NewServeMux()
	mux.Handle(cfg.GetMetricsRoute(), promhttp.Handler())

	for _, opt := range options {
		mux = opt(mux)
	}

	return runMetricsServer(logger, cfg, mux)
}

func runMetricsServer(logger logging.Logger, cfg Config, mux *http.ServeMux) Server {
	server := NewServerFromConfig(cfg, mux)
	go func() {
		errChan := server.Start()
		for err := range errChan {
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error(
					"Error occurred serving the metrics endpoint",
					zap.Error(err),
				)
			}
		}
	}()
	return server
}
