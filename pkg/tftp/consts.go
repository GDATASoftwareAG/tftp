package tftp

type ErrorCode uint16

func (ec ErrorCode) Is(other ErrorCode) bool {
	return ec == other
}
func (ec ErrorCode) Value() uint16 {
	return uint16(ec)
}

const (
	OACKBlockNumber = 0
)

const (
	// Error codes
	// http://tools.ietf.org/html/rfc1350#page-10
	UnknownError     ErrorCode = iota // Not defined, see error message (if any).
	NotFound                          // File not found.
	AccessViolation                   // Access violation.
	DiskFull                          // Disk full or allocation exceeded.
	IllegalOperation                  // Illegal TFTP operation.
	UnknownID                         // Unknown transfer ID.
	AlreadyExists                     // File already exists.
	NoSuchUser                        // No such user.
)

func (e ErrorCode) String() string {
	switch e {
	case NotFound:
		return "NotFound"
	case AccessViolation:
		return "AccessViolation"
	case DiskFull:
		return "DiskFull"
	case IllegalOperation:
		return "IllegalOperation"
	case UnknownID:
		return "UnknownID"
	case AlreadyExists:
		return "AlreadyExists"
	case NoSuchUser:
		return "NoSuchUser"
	default:
		return "UnknownError"
	}
}

type OpcodeServer uint16

func (o OpcodeServer) Value() uint16 {
	return uint16(o)
}

type OpcodeClient uint16

func (oc OpcodeClient) Is(other OpcodeClient) bool {
	return oc == other
}
func (o OpcodeClient) Value() uint16 {
	return uint16(o)
}

func (o OpcodeServer) String() string {
	switch o {
	case DATA:
		return "DATA"
	case ServerError:
		return "ServerError"
	case OACK:
		return "OACK"
	default:
		return "unknown server opcode"
	}
}

func (o OpcodeClient) String() string {
	switch o {
	case RRQ:
		return "RRQ"
	case WRQ:
		return "WRQ"
	case OACKFileTooLarge:
		return "OACKFileTooLarge"
	case ACK:
		return "ACK"
	case ClientError:
		return "ClientError"
	case OACKDeclined:
		return "OACKDeclined"
	default:
		return "unknown client opcode"
	}
}

const (
	RRQ              OpcodeClient = 1
	WRQ              OpcodeClient = 2
	OACKFileTooLarge OpcodeClient = 3
	ACK              OpcodeClient = 4
	ClientError      OpcodeClient = 5
	OACKDeclined     OpcodeClient = 8

	DATA        OpcodeServer = 3
	ServerError OpcodeServer = 5
	OACK        OpcodeServer = 6
)

const (
	FilepathFieldIndex    int = 0
	ModeFieldIndex        int = 1
	FirstOptionFieldIndex int = 2
)

const (
	ReadBufferSize = 1024

	OpcodeLength   = 2
	BlocknumLength = 2
	MetaInfoSize   = OpcodeLength + BlocknumLength

	DefaultBlocksize    = 512
	DefaultTransfersize = -1

	// modes
	ModeOctet    = "octet"
	ModeNetAscii = "netascii"
	Octet        = 1
	NetAscii     = 2

	OptionBlksize = "blksize"
	OptionTsize   = "tsize"
)
