package tftp

import "github.com/gdatasoftwareag/tftp/internal/metrics"

type Config struct {
	IP                     string
	Port                   int
	Retransmissions        int
	MaxParallelConnections int

	Metrics metrics.Config
}
