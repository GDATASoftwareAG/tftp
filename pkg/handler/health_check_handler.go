package handler

import (
	"io"
	"strings"

	"github.com/gdatasoftwareag/tftp/pkg/tftp"
)

const (
	HealthCheckContent = "health"
)

func NewHealthCheckHandler() tftp.Handler {
	return &healthCheckHandler{}
}

type healthCheckHandler struct {
}

func (h *healthCheckHandler) Matches(file string) bool {
	return strings.ToLower(file) == HealthCheckContent
}

func (h *healthCheckHandler) Reader(_ string) (io.ReadCloser, int64, error) {
	return io.NopCloser(strings.NewReader(HealthCheckContent)), int64(len(HealthCheckContent)), nil
}
