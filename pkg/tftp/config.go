package tftp

import (
	"github.com/gdatasoftwareag/tftp/v2/internal/metrics"
	"time"
)

const (
	defaultWriteTimeout = 5 * time.Second
	defaultReadTimeout  = 5 * time.Second
)

type Config struct {
	IP                     string
	Port                   int
	Retransmissions        int
	MaxParallelConnections int
	FileTransferTimeout    time.Duration
	WriteTimeout           time.Duration
	ReadTimeout            time.Duration
	Metrics                metrics.Config
}

func (c Config) ReadTimeoutOrDefault() time.Duration {
	if c.ReadTimeout == 0 {
		return defaultReadTimeout
	}
	return c.ReadTimeout
}

func (c Config) WriteTimeoutOrDefault() time.Duration {
	if c.WriteTimeout == 0 {
		return defaultWriteTimeout
	}
	return c.WriteTimeout
}
