package tftp

import (
	"encoding/binary"
	"reflect"
	"strconv"
	"testing"
)

const (
	filename string = "foo.txt"
)

type Option struct {
	Name string
	Val  string
}

func CreateRRQTestData(path string, mode string, options []Option) []byte {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, RRQ.Value())
	data = append(data, []byte(path)...)
	data = append(data, 0)
	data = append(data, []byte(mode)...)
	data = append(data, 0)

	for _, opt := range options {
		switch opt.Name {
		case OptionBlksize:
			if opt.Val != strconv.Itoa(DefaultBlocksize) {
				data = setOACKOption(data, opt.Name, opt.Val)
			}
		case OptionTsize:
			if opt.Val != strconv.Itoa(DefaultTransfersize) {
				data = setOACKOption(data, opt.Name, opt.Val)
			}
		default:
			data = setOACKOption(data, opt.Name, opt.Val)
		}
	}
	return data
}

func TestParseRequest(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *Request
		wantErr bool
	}{
		{
			name: "Test request with default values",
			args: args{
				data: CreateRRQTestData(filename, ModeOctet, nil),
			},
			want: &Request{
				Opcode:       RRQ,
				Blocksize:    DefaultBlocksize,
				TransferSize: DefaultTransfersize,
				Mode:         Octet,
				Path:         filename,
			},
			wantErr: false,
		},
		{
			name: "Test request with custom blocksize",
			args: args{
				data: CreateRRQTestData(filename, ModeOctet, []Option{
					Option{
						Name: OptionBlksize,
						Val:  "123",
					}},
				),
			},
			want: &Request{
				Opcode:       RRQ,
				Blocksize:    123,
				TransferSize: DefaultTransfersize,
				Mode:         Octet,
				Path:         filename,
			},
			wantErr: false,
		},
		{
			name: "Test request with transfersize",
			args: args{
				data: CreateRRQTestData(filename, ModeOctet, []Option{
					Option{
						Name: OptionTsize,
						Val:  "0",
					}},
				),
			},
			want: &Request{
				Opcode:       RRQ,
				Blocksize:    DefaultBlocksize,
				TransferSize: 0,
				Mode:         Octet,
				Path:         filename,
			},
			wantErr: false,
		},
		{
			name: "Test request with transfersize and custom blocksize",
			args: args{
				data: CreateRRQTestData(filename, ModeOctet, []Option{
					Option{
						Name: OptionTsize,
						Val:  "0",
					}, Option{
						Name: OptionBlksize,
						Val:  "123",
					}},
				),
			},
			want: &Request{
				Opcode:       RRQ,
				Blocksize:    123,
				TransferSize: 0,
				Mode:         Octet,
				Path:         filename,
			},
			wantErr: false,
		},
		{
			name: "Test request with netascii encoding",
			args: args{
				data: CreateRRQTestData(filename, ModeNetAscii, nil),
			},
			want: &Request{
				Opcode:       RRQ,
				Blocksize:    DefaultBlocksize,
				TransferSize: DefaultTransfersize,
				Mode:         NetAscii,
				Path:         filename,
			},
			wantErr: false,
		},
		{
			name: "Test request with unknown options",
			args: args{
				data: CreateRRQTestData(filename, ModeOctet, []Option{
					Option{
						Name: "Weird",
						Val:  "Option",
					}}),
			},
			want: &Request{
				Opcode:       RRQ,
				Blocksize:    DefaultBlocksize,
				TransferSize: DefaultTransfersize,
				Mode:         Octet,
				Path:         filename,
			},
			wantErr: false,
		},
		{
			name: "Test request with unknown and known options",
			args: args{
				data: CreateRRQTestData(filename, ModeOctet, []Option{
					Option{
						Name: OptionBlksize,
						Val:  "234",
					}, Option{
						Name: "Weird",
						Val:  "Option",
					}}),
			},
			want: &Request{
				Opcode:       RRQ,
				Blocksize:    234,
				TransferSize: DefaultTransfersize,
				Mode:         Octet,
				Path:         filename,
			},
			wantErr: false,
		},
		{
			name: "Test request with invalid mode",
			args: args{
				data: CreateRRQTestData(filename, "WrongMode", nil),
			},
			wantErr: true,
		},
		{
			name: "Test request with invalid opcode",
			args: args{
				data: func() []byte {
					data := make([]byte, 2)
					binary.BigEndian.PutUint16(data, ServerError.Value())
					return data
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRequest(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
