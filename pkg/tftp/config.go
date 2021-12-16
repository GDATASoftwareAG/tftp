package tftp

import (
	"github.com/gdatasoftwareag/tftp/v2/internal/metrics"
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
