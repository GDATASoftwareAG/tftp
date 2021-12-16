package tftp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/gdatasoftwareag/tftp/v2/pkg/udp"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/gdatasoftwareag/tftp/v2/pkg/logging"
	"go.uber.org/zap"
)

type ackValue int

const (
	OK ackValue = iota
	Retransmit
	Error
	Failure
)

var (
	sendFileCounter *prometheus.CounterVec
	ackBufferPool   = sync.Pool{
		New: func() interface{} {
			return make([]byte, ReadBufferSize)
		},
	}
)

func init() {
	sendFileCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tftp_send_file_total",
		Help: "Total number of transmitted files - the result label indicates whether to file was submitted successfully or if any kind of TFTP error occured during the transmission",
	}, []string{"result"})

	prometheus.MustRegister(sendFileCounter)
}

type oackOption struct {
	name string
	val  string
}

type rrqResponse struct {
	logger           logging.Logger
	responseHandling ResponseHandling
	connection       udp.Connection
	retransmissions  int
	ack              []byte
	ackLength        int
	request          *Request
	blocknum         uint16
	blocksize        int
	transfersize     int64
}

func (r *rrqResponse) SendFile(ctx context.Context) error {
	r.logger.Info("Trying to send file to client",
		zap.String("File", r.request.Path))
	reader, size, err := r.responseHandling.OpenFile(ctx, r.request.Path)
	r.transfersize = size
	if err != nil {
		r.handleSendFileError(err)
		return err
	}
	r.logger.Debug("Successfully opened file")

	if required, options := r.determineRequiredOACKOptions(); required {
		if err = r.sendOACK(options); err != nil {
			return err
		}
	}

	sendFileCounter.WithLabelValues("Success").Inc()
	return r.sendTFTPBlocks(ctx, reader)
}

func (r *rrqResponse) handleSendFileError(err error) {
	var errorCode ErrorCode
	if errors.Is(err, FileErrors.ErrPermission) {
		errorCode = AccessViolation
	} else if errors.Is(err, FileErrors.ErrNotFound) || errors.Is(err, FileErrors.ErrReadFile) {
		errorCode = NotFound
	} else if errors.Is(err, FileErrors.ErrInsecure) {
		errorCode = IllegalOperation
	} else {
		errorCode = UnknownError
	}

	sendFileCounter.WithLabelValues(errorCode.String()).Inc()
	if sendErr := r.sendError(errorCode, err); sendErr != nil {
		r.logger.Error("failed to send error", zap.Error(err))
	}
	r.logger.Error("Could not open file",
		zap.Error(err),
	)
}

func (r *rrqResponse) sendTFTPBlocks(ctx context.Context, reader io.ReadCloser) (err error) {
	defer func() {
		if err := reader.Close(); err != nil {
			r.logger.Warn("Failed to close reader", zap.Error(err))
		}
	}()

	buffer := make([]byte, r.blocksize+MetaInfoSize)
	r.logger.Debug("Sending blocks to client")

	currentBlock, ok := r.createTFTPBlock(reader, buffer, r.blocknum)

	for ok && ctx.Err() == nil {
		if err = r.sendTFTPBlock(currentBlock); err != nil {
			return
		}

		r.blocknum++
		currentBlock, ok = r.createTFTPBlock(reader, buffer, r.blocknum)
	}

	if r.transfersize%int64(r.blocksize) == 0 {
		err = r.sendTFTPBlock(currentBlock)
	}

	return
}

func (r *rrqResponse) sendTFTPBlock(block []byte) (err error) {
	for transmissionCount := 1; transmissionCount <= r.retransmissions; transmissionCount++ {
		if err = r.sendBufferWithRetransmissionOnTimeout(block, r.retransmissions); err != nil {
			return
		}

		var ackResult ackValue
		ackResult, err = r.parseACK(r.blocknum)
		switch ackResult {
		case OK:
			return
		case Retransmit:
			if transmissionCount >= 3 {
				return fmt.Errorf("aborting transmission after %d retries", r.retransmissions)
			}
			r.logger.Debug("Retransmitting block",
				zap.Int("Retransmission", transmissionCount))
			continue
		case Error:
			err = fmt.Errorf("received error from client - %w", err)
		case Failure:
			err = fmt.Errorf("received unexpected response from client - %w", err)
		}
	}
	return
}

func (r *rrqResponse) createTFTPBlock(reader io.Reader, buffer []byte, blocknum uint16) ([]byte, bool) {
	read, err := reader.Read(buffer[MetaInfoSize:])
	binary.BigEndian.PutUint16(buffer, DATA.Value())
	binary.BigEndian.PutUint16(buffer[OpcodeLength:], blocknum)

	if read == 0 && err != nil {
		return buffer[:MetaInfoSize], false
	}

	return buffer[:read+MetaInfoSize], true
}

func (r *rrqResponse) determineRequiredOACKOptions() (required bool, options []oackOption) {
	if r.request.Blocksize != DefaultBlocksize {
		options = append(options, oackOption{name: OptionBlksize, val: strconv.Itoa(r.request.Blocksize)})
	}

	if r.request.TransferSize != DefaultTransfersize {
		options = append(options, oackOption{name: OptionTsize, val: strconv.FormatInt(r.transfersize, 10)})
	}

	return len(options) > 0, options
}

func setOACKOption(buffer []byte, option string, value string) []byte {
	buffer = append(buffer, []byte(option)...)
	buffer = append(buffer, 0)
	buffer = append(buffer, []byte(value)...)
	buffer = append(buffer, 0)
	return buffer
}

func (r *rrqResponse) sendOACK(options []oackOption) (err error) {
	r.logger.Debug("Sending OACK",
		zap.Int("blocksize", r.request.Blocksize),
		zap.Int64("transfersize", r.transfersize))

	oackbuffer := make([]byte, OpcodeLength)
	binary.BigEndian.PutUint16(oackbuffer, OACK.Value())

	for _, opt := range options {
		oackbuffer = setOACKOption(oackbuffer, opt.name, opt.val)
	}

	if err = r.sendBufferWithRetransmissionOnTimeout(oackbuffer, 1); err != nil {
		return
	}

	opcode := OpcodeClient(binary.BigEndian.Uint16(r.ack))
	if opcode.Is(ACK) && OACKBlockNumber == binary.BigEndian.Uint16(r.ack[OpcodeLength:]) {
		r.blocksize = r.request.Blocksize
	} else if opcode.Is(OACKDeclined) {
		err = ReceiveError.ErrDeclinedByClient
		r.logger.Warn(err.Error())
	} else if opcode.Is(OACKFileTooLarge) {
		err = ReceiveError.ErrFileTooLarge
		r.logger.Warn(err.Error())
	} else {
		err = ReceiveError.ErrWeirdOpcode
		r.logger.Warn(err.Error(),
			zap.Uint16("Opcode", binary.BigEndian.Uint16(r.ack)))
	}

	return
}

func (r *rrqResponse) parseACK(expectedBlockNum uint16) (ackValue, error) {
	opcode := OpcodeClient(binary.BigEndian.Uint16(r.ack))

	if opcode.Is(ClientError) {
		return Error, fmt.Errorf("message: %s: %w",
			string(r.ack[MetaInfoSize:r.ackLength-1]), ReceiveError.ErrClient)
	}
	if !opcode.Is(ACK) {
		return Failure, fmt.Errorf("ACK: %s: %w", opcode, ReceiveError.ErrWeirdOpcode)
	}

	acknum := binary.BigEndian.Uint16(r.ack[OpcodeLength:])
	if acknum == expectedBlockNum-1 {
		return Retransmit, nil
	}

	if acknum != expectedBlockNum {
		return Failure, fmt.Errorf(
			"ACK num %v, expected %v: %w",
			acknum,
			expectedBlockNum,
			ReceiveError.ErrWeirdBlocknum,
		)
	}

	return OK, nil
}

func (r *rrqResponse) sendError(code ErrorCode, occurredError error) (err error) {

	msg := occurredError.Error()

	// http://tools.ietf.org/html/rfc1350#page-8
	errorbuffer := make([]byte, 2+2+len(msg)+1)

	binary.BigEndian.PutUint16(errorbuffer, ServerError.Value())
	binary.BigEndian.PutUint16(errorbuffer[OpcodeLength:], code.Value())

	copy(errorbuffer[MetaInfoSize:], msg)
	errorbuffer[len(errorbuffer)-1] = 0

	_, err = r.connection.Write(errorbuffer)
	return
}

func (r *rrqResponse) sendBufferWithRetransmissionOnTimeout(buffer []byte, tries int) error {
	var err error
	var numBytes int

	_ = r.connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
	numBytes, err = r.connection.Write(buffer)
	if err != nil {
		return err
	}

	r.logger.Debug("Package send",
		zap.Int("Bytes", numBytes))

	for try := 0; try < tries; try++ {
		_ = r.connection.SetReadDeadline(time.Now().Add(5 * time.Second))
		numBytes, _, err = r.connection.ReadFromUDP(r.ack)
		r.ackLength = numBytes
		if err == nil {
			r.logger.Debug("Package received",
				zap.Int("Bytes", numBytes),
				zap.ByteString("bytes", r.ack[:numBytes]))
			return nil
		}
		r.logger.Warn("Read timeout",
			zap.Int("Try", try))
	}

	return err
}

func newRRQResponse(responseHandling ResponseHandling,
	connection udp.Connection,
	request *Request,
	retransmissions int,
	logger logging.Logger) *rrqResponse {
	return &rrqResponse{
		logger:           logger,
		responseHandling: responseHandling,
		connection:       connection,
		retransmissions:  retransmissions,
		ack:              ackBufferPool.Get().([]byte),
		request:          request,
		blocknum:         1,
		blocksize:        DefaultBlocksize,
	}
}
