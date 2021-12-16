package tftp

import (
	"github.com/gdatasoftwareag/tftp/internal/metrics"
	"time"
)

type Config struct {
	IP                     string
	Port                   int
	Retransmissions        int
	MaxParallelConnections int
	FileTransferTimeout    time.Duration

	Metrics metrics.Config
}
