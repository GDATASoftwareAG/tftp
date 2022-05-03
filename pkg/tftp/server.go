package tftp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gdatasoftwareag/tftp/v2/internal/metrics"
	"github.com/gdatasoftwareag/tftp/v2/pkg/udp"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/multierr"

	"github.com/gdatasoftwareag/tftp/v2/pkg/logging"
	"go.uber.org/zap"
)

var (
	requestCounter          prometheus.Counter
	requestLatencyHistogram prometheus.Histogram
)

func init() {
	requestCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tftp_requests_total",
		Help: "Total number of TFTP requests handled by the server during its lifetime",
	})
	requestLatencyHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "tftp_send_file_latency",
		Help: "Latency histogram how long it took to completely send a file",
	})

	prometheus.MustRegister(requestCounter, requestLatencyHistogram)
}

type Server interface {
	ListenAndServe(ctx context.Context) error
}

type server struct {
	logger                  logging.Logger
	cfg                     Config
	connector               udp.Connector
	listenAddr              *net.UDPAddr
	responseHandling        ResponseHandling
	clientRequestBufferPool *sync.Pool
	metricsServer           metrics.Server
	rrqChan                 chan *Request
}

func NewServer(logger logging.Logger, config Config, udpConnector udp.Connector, responseHandling ResponseHandling) (Server, error) {
	var metricsServer metrics.Server
	if config.Metrics.Enabled {
		metricsServer = metrics.StartMetricsServer(logger, config.Metrics, metrics.WithProfilingHooks())
	}

	return &server{
		logger:           logger,
		cfg:              config,
		connector:        udpConnector,
		listenAddr:       &net.UDPAddr{IP: parseIPOrFallback(config.IP, logger), Port: config.Port},
		responseHandling: responseHandling,
		clientRequestBufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, ReadBufferSize)
			},
		},
		metricsServer: metricsServer,
	}, nil
}

func (s *server) ListenAndServe(ctx context.Context) (err error) {
	var listenSocket udp.Connection
	if listenSocket, err = s.connector.ListenUDP("udp", s.listenAddr); err != nil {
		return
	}

	s.rrqChan = make(chan *Request)
	defer close(s.rrqChan)

	for workerIdx := 0; workerIdx < s.cfg.MaxParallelConnections; workerIdx++ {
		go s.rrqWorker(ctx)
	}
	errChan := s.listen(ctx, listenSocket)

	for {
		select {
		case <-ctx.Done():
			if s.metricsServer != nil {
				err = s.metricsServer.Shutdown(5 * time.Second)
			}
			s.logger.Info("Closing socket")
			err = multierr.Append(err, listenSocket.Close())
			return
		case e := <-errChan:
			if e != nil {
				s.logger.Error("Message accept encountered an error", zap.Error(e))
			}
		}
	}
}

func (s server) listen(ctx context.Context, listenSocket udp.Connection) <-chan error {
	errChan := make(chan error)

	go func() {
		defer close(errChan)
		for ctx.Err() == nil {
			if err := s.accept(listenSocket); err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
}

func (s *server) accept(listenSocket udp.Connection) (err error) {
	var written int
	var addr *net.UDPAddr

	requestBuffer := s.clientRequestBufferPool.Get().([]byte)
	written, addr, err = listenSocket.ReadFromUDP(requestBuffer)
	requestCounter.Inc()
	if err != nil {
		return fmt.Errorf("failed to read data from client: %v", err)
	}

	s.logger.Info("Received request",
		zap.Any("Addr", addr))

	var request *Request
	request, err = parseRequest(requestBuffer[:written])
	if err != nil {
		return fmt.Errorf("failed to parse request: %v", err)
	} else {
		s.logger.Info("Parsed request",
			zap.String("Request", request.ToString()))
	}

	if !request.Opcode.Is(RRQ) {
		s.logger.Warn("Unknown Opcode",
			zap.Uint16("Opcode", request.Opcode.Value()))
		return
	}

	request.ClientAddress = addr
	s.rrqChan <- request

	return
}

func (s *server) rrqWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case request, more := <-s.rrqChan:
			if !more {
				return
			}

			var sendCtx context.Context
			var cancel context.CancelFunc

			if s.cfg.FileTransferTimeout != 0 {
				sendCtx, cancel = context.WithTimeout(ctx, s.cfg.FileTransferTimeout)
			} else {
				sendCtx, cancel = context.WithCancel(ctx)
			}

			err := s.sendFile(sendCtx, request)
			cancel()

			if err != nil {
				s.logger.Error("Failed to send file",
					zap.Any("clientAddr", request.ClientAddress),
					zap.String("file", request.Path),
					zap.Error(err))
			}
		}
	}
}

func (s *server) sendFile(ctx context.Context, request *Request) error {
	s.logger.Info("Sending file...", zap.Any("clientAddr", request.ClientAddress), zap.String("File", request.Path))
	sendFileTimer := prometheus.NewTimer(requestLatencyHistogram)
	defer sendFileTimer.ObserveDuration()

	s.logger.Debug("Establish fileTransferConnection")
	connection, err := s.getConnection(request.ClientAddress)
	if err != nil {
		return err
	}
	defer connection.Close()

	response := newRRQResponse(s.responseHandling, connection, request, s.cfg, s.logger)
	if err = response.SendFile(ctx); err != nil {
		return err
	}

	s.logger.Info("File transmitted successfully")
	return nil
}

func (s *server) getConnection(clientAddr *net.UDPAddr) (udp.Connection, error) {
	// With the port 0 the library is advised to pick a random free high port
	listenAddr, err := s.connector.ResolveUDPAddr("udp", fmt.Sprintf("%s:0", s.listenAddr.IP.String()))
	if err != nil {
		return nil, err
	}

	conn, err := s.connector.DialUDP("udp", listenAddr, clientAddr)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Open fileTransferConnection",
		zap.Any("ListenAddr", conn.LocalAddr()),
		zap.Any("ClientAddr", clientAddr),
	)

	return conn, nil
}

func parseIPOrFallback(input string, logger logging.Logger) (ip net.IP) {
	if ip = net.ParseIP(input); ip == nil {
		logger.Warn("IP invalid, using fallback 0.0.0.0",
			zap.String("IP", input))
		ip = net.ParseIP("0.0.0.0")
	}
	return
}
