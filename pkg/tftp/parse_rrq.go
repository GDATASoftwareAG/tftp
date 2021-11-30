package tftp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

var NullByte = []byte{0}

type Request struct {
	Opcode        OpcodeClient
	Blocksize     int
	TransferSize  int64
	Mode          int
	Path          string
	ClientAddress *net.UDPAddr
}

type RRQParseError struct {
	msg string
}

func (e *RRQParseError) Error() string {
	return e.msg
}

func parseRequest(data []byte) (*Request, error) {
	request := &Request{
		Blocksize:    DefaultBlocksize,
		Opcode:       OpcodeClient(binary.BigEndian.Uint16(data)),
		TransferSize: DefaultTransfersize,
	}

	if !request.Opcode.Is(RRQ) {
		return nil, fmt.Errorf("opcode: %d: %w", request.Opcode, RequestErrors.ErrUnknownOption)
	}

	rest := data[OpcodeLength:]

	dataSlice := bytes.Split(rest, NullByte)

	request.Path = string(dataSlice[FilepathFieldIndex])

	if err := request.SetMode(string(dataSlice[ModeFieldIndex])); err != nil {
		return request, fmt.Errorf("Mode: %d: %w", request.Mode, RequestErrors.ErrUnknownMode)
	}

	for idxOption := FirstOptionFieldIndex; idxOption < len(dataSlice); idxOption += 2 {
		option := dataSlice[idxOption]

		if len(option) == 0 {
			break
		}
		if idxOption+1 == len(dataSlice) {
			return nil, RequestErrors.ErrMalformed
		}

		if err := request.SetOptions(string(option), string(dataSlice[idxOption+1])); err != nil {
			return request, err
		}
	}

	return request, nil
}

func (r *Request) SetMode(mode string) error {
	switch mode {
	case ModeOctet:
		r.Mode = Octet
	case ModeNetAscii:
		r.Mode = NetAscii
	default:
		return fmt.Errorf("mode: %d: %w", r.Mode, RequestErrors.ErrUnknownMode)
	}
	return nil
}

func (r *Request) SetOptions(option, value string) error {
	switch option {
	case OptionBlksize:
		if blocksize, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("blksize: %s: %w", value, RequestErrors.ErrInvalidBlksize)
		} else {
			r.Blocksize = blocksize
		}
	case OptionTsize:
		if tsize, err := strconv.ParseInt(value, 10, 0); err != nil {
			return fmt.Errorf("tsize: %s: %w", value, RequestErrors.ErrInvalidTransfersize)
		} else {
			r.TransferSize = tsize
		}
	default:
		return nil
	}
	return nil
}

func (r *Request) ToString() string {
	return fmt.Sprintf("Opcode: %d\nBlocksize: %d\nTransferSize: %d, Mode: %d\nPath: %s\n",
		r.Opcode,
		r.Blocksize,
		r.TransferSize,
		r.Mode,
		r.Path)
}
