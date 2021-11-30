package tftp

import "errors"

var (
	ReceiveError = struct {
		ErrDeclinedByClient error
		ErrFileTooLarge     error
		ErrClient           error
		ErrWeirdBlocknum    error
		ErrWeirdOpcode      error
	}{
		ErrDeclinedByClient: errors.New("oack was declined by the client"),
		ErrFileTooLarge:     errors.New("file was too large for client"),
		ErrClient:           errors.New("received error from client"),
		ErrWeirdBlocknum:    errors.New("received unexpected blocknum from client"),
		ErrWeirdOpcode:      errors.New("client sent unexpected opcode"),
	}
)
var (
	SendErrors = struct {
		ErrAbort  error
		ErrServer error
	}{
		ErrAbort:  errors.New("aborting tranmission after too many retries"),
		ErrServer: errors.New("transmission error in the server"),
	}
)

var (
	FileErrors = struct {
		ErrNoHandler  error
		ErrInsecure   error
		ErrReadFile   error
		ErrPermission error
		ErrNotFound   error
	}{
		ErrNoHandler:  errors.New("no handler found for file"),
		ErrInsecure:   errors.New("file path was not valid"),
		ErrReadFile:   errors.New("error during read of file"),
		ErrPermission: errors.New("insufficient permission to read file"),
		ErrNotFound:   errors.New("file not found"),
	}
)

var (
	RequestErrors = struct {
		ErrUnknownOpcode       error
		ErrUnknownMode         error
		ErrUnknownOption       error
		ErrInvalidBlksize      error
		ErrInvalidTransfersize error
		ErrMalformed           error
	}{
		ErrUnknownOpcode:       errors.New("unknown opcode in request"),
		ErrUnknownMode:         errors.New("unknown mode in request"),
		ErrUnknownOption:       errors.New("unknown option in request"),
		ErrInvalidBlksize:      errors.New("invalid blocksize in request"),
		ErrInvalidTransfersize: errors.New("invalid transfersize in request"),
		ErrMalformed:           errors.New("received malformed request"),
	}
)
