package tftp_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gdatasoftwareag/tftp/v2/pkg/handler"
	"github.com/gdatasoftwareag/tftp/v2/pkg/logging"
	"github.com/gdatasoftwareag/tftp/v2/pkg/secfs"
	"github.com/gdatasoftwareag/tftp/v2/pkg/tftp"
	"github.com/gdatasoftwareag/tftp/v2/pkg/udp"
	"github.com/gdatasoftwareag/tftp/v2/pkg/udp/mock"
	"github.com/golang/mock/gomock"
)

var listenAddress = &net.UDPAddr{
	IP:   net.ParseIP("0.0.0.0"),
	Port: 63,
}
var resolvedListenAddress = &net.UDPAddr{IP: listenAddress.IP, Port: 0}
var clientAddress = &net.UDPAddr{
	IP:   net.ParseIP("69.69.69.69"),
	Port: 420,
}

func TestServer_ListenAndServe(t *testing.T) {
	t.Parallel()
	//Set discard to false to see all logs from the tests
	testLogger := logging.CreateTestLogger(t, true)

	type fields struct {
		tftpFlowControlFunc func(ctrl *gomock.Controller) *TFTPFlowControl
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Request file smaller than blocksize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, "smallFile.txt").
						createTFTPBlocksFromBuffer(readFileToBuffer("smallFile.txt"))

					return controller
				},
			},
			wantErr: false,
		},
		{
			name: "Request file equals blocksize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, "defaultBlockSizeFile.txt").
						createTFTPBlocksFromBuffer(readFileToBuffer("defaultBlockSizeFile.txt"))

					return controller
				},
			},
			wantErr: false,
		},
		{
			name: "Request file larger blocksize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, "largeFile.txt").
						createTFTPBlocksFromBuffer(readFileToBuffer("largeFile.txt"))

					return controller
				},
			},
			wantErr: false,
		},
		{
			name: "Test OACK with different block- and transfersize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					fileBuffer := readFileToBuffer("defaultBlockSizeFile.txt")
					options := []tftp.Option{
						{
							Name: tftp.OptionBlksize,
							Val:  "123",
						},
						{
							Name: tftp.OptionTsize,
							Val:  "0",
						}}
					controller := setupFileTransferConnections(ctrl, "defaultBlockSizeFile.txt", options...).
						createOACK(123, len(fileBuffer)).
						createTFTPBlocksFromBuffer(fileBuffer)

					return controller
				},
			},
			wantErr: false,
		},
		{
			name: "Test OACK with different blocksize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					fileBuffer := readFileToBuffer("defaultBlockSizeFile.txt")
					options := []tftp.Option{
						{
							Name: tftp.OptionBlksize,
							Val:  "123",
						},
						{
							Name: tftp.OptionTsize,
							Val:  strconv.Itoa(tftp.DefaultTransfersize),
						}}
					controller := setupFileTransferConnections(ctrl, "defaultBlockSizeFile.txt", options...).
						createOACK(123, tftp.DefaultTransfersize).
						createTFTPBlocksFromBuffer(fileBuffer)

					return controller
				},
			},
			wantErr: false,
		},
		{
			name: "Test OACK with different transfersize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					fileBuffer := readFileToBuffer("defaultBlockSizeFile.txt")
					options := []tftp.Option{
						{
							Name: tftp.OptionBlksize,
							Val:  strconv.Itoa(tftp.DefaultBlocksize),
						},
						{
							Name: tftp.OptionTsize,
							Val:  "0",
						}}
					controller := setupFileTransferConnections(ctrl, "defaultBlockSizeFile.txt", options...).
						createOACK(tftp.DefaultBlocksize, len(fileBuffer)).
						createTFTPBlocksFromBuffer(fileBuffer)

					return controller
				},
			},
			wantErr: false,
		},
		{
			name: "Test file not found",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, "not_present_file.txt").
						createError(tftp.NotFound)

					return controller
				},
			},
			wantErr: true,
		},
		{
			name: "Test read timeout",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, "defaultBlockSizeFile.txt").
						createReadTimeout()

					return controller
				},
			},
			wantErr: true,
		},
		{
			name: "Test write timeout",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, "smallFile.txt").
						createWriteTimeout()

					return controller
				},
			},
			wantErr: true,
		},
		{
			name: "Test retransmission of package",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					fileBuffer := readFileToBuffer("smallFile.txt")
					controller := setupFileTransferConnections(ctrl, "smallFile.txt").
						createRetransmission(fileBuffer, 1, true)

					return controller
				},
			},
			wantErr: false,
		},
		{
			name: "Test retransmission failure of package",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					fileBuffer := readFileToBuffer("smallFile.txt")
					controller := setupFileTransferConnections(ctrl, "smallFile.txt").
						createRetransmission(fileBuffer, 3, false)

					return controller
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			dir, err := os.Getwd()
			if err != nil {
				t.Errorf("Couldn't get working directory: %s", err)
			}
			dir = filepath.Join(dir, "test_files")

			responseHandling := tftp.NewResponseHandling(testLogger)
			responseHandling.RegisterHandler(handler.NewFSHandler(secfs.New(dir), testLogger))

			config := tftp.Config{
				IP:                     "0.0.0.0",
				Port:                   63,
				Retransmissions:        3,
				MaxParallelConnections: 10,
				FileTransferTimeout:    10 * time.Second,
			}

			flowControl := tt.fields.tftpFlowControlFunc(ctrl)
			server, err := tftp.NewServer(testLogger,
				config,
				flowControl.connector,
				responseHandling)
			if err != nil {
				t.Errorf("Failed to create TFTP Server: err = %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			doneChan := make(chan struct{})
			go func() {
				_ = server.ListenAndServe(ctx)
				close(doneChan)
			}()
			<-flowControl.transmissionDone
			cancel()
			<-doneChan

		})
	}
}

func readFileToBuffer(path string) []byte {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	path = filepath.Join(dir, "test_files", path)
	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return fileContent
}

type TFTPFlowControl struct {
	connector              *mock.MockConnector
	fileTransferConnection *mock.MockConnection
	blockSize              int
	blockNum               uint16
	transmissionDone       chan struct{}
}

func createUDPConnectionWithOneIncomingRequest(ctrl *gomock.Controller, file string, requestOptions ...tftp.Option) udp.Connection {
	listenForRRQConnection := mock.NewMockConnection(ctrl)
	// Return valid RRQ on first read.
	// Simulate closed connection on all following reads. Test cases only handle one incoming request.
	gomock.InOrder(
		listenForRRQConnection.
			EXPECT().
			ReadFromUDP(gomock.Any()).
			DoAndReturn(func(b []byte) (_ int, _ *net.UDPAddr, _ error) {
				rrqData := tftp.CreateRRQTestData(file, tftp.ModeOctet, requestOptions)
				copy(b, rrqData)
				return len(rrqData), clientAddress, nil
			}),
		listenForRRQConnection.
			EXPECT().
			ReadFromUDP(gomock.Any()).
			DoAndReturn(func(b []byte) (_ int, _ *net.UDPAddr, _ error) {
				return 0, clientAddress, io.EOF
			}).AnyTimes(),
	)
	listenForRRQConnection.EXPECT().Close()

	return listenForRRQConnection
}

func createFileTransferConnection(ctrl *gomock.Controller, transmissionDoneChan chan struct{}) *mock.MockConnection {
	mockConnection := mock.NewMockConnection(ctrl)
	mockConnection.EXPECT().Close().Do(func() error {
		close(transmissionDoneChan)
		return nil
	})

	// Called for every sending/receiving packet
	mockConnection.EXPECT().SetWriteDeadline(gomock.Any()).AnyTimes()
	mockConnection.EXPECT().SetReadDeadline(gomock.Any()).AnyTimes()

	// Only called for logging
	mockConnection.
		EXPECT().
		LocalAddr().
		Return(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 63234}).
		AnyTimes()

	return mockConnection
}

func setupFileTransferConnections(ctrl *gomock.Controller, file string, requestOptions ...tftp.Option) *TFTPFlowControl {
	mockConnector := mock.NewMockConnector(ctrl)
	mockConnector.
		EXPECT().
		ListenUDP("udp", gomock.Any()).
		Return(createUDPConnectionWithOneIncomingRequest(ctrl, file, requestOptions...), nil)

	// Server establishes connection to incoming request
	//    Get a local address with random high port
	mockConnector.EXPECT().
		ResolveUDPAddr("udp", listenAddress.IP.String()+":0").
		Return(resolvedListenAddress, nil)

	//    Establish connection for file transfer
	transmissionDoneChan := make(chan struct{})
	fileTransferConnection := createFileTransferConnection(ctrl, transmissionDoneChan)
	mockConnector.
		EXPECT().
		DialUDP("udp", resolvedListenAddress, clientAddress).
		Return(fileTransferConnection, nil)

	return &TFTPFlowControl{
		connector:              mockConnector,
		fileTransferConnection: fileTransferConnection,
		blockSize:              tftp.DefaultBlocksize,
		blockNum:               1,
		transmissionDone:       transmissionDoneChan,
	}
}

func (t *TFTPFlowControl) createTFTPBlocksFromBuffer(buffer []byte) *TFTPFlowControl {
	end := int(math.Ceil(float64(len(buffer)) / float64(t.blockSize)))

	for idx := 0; idx < end; idx++ {
		var tftpBlock []byte
		if (idx+1)*t.blockSize <= len(buffer) {
			tftpBlock = createTFTPBlock(buffer[idx*t.blockSize:(idx+1)*t.blockSize], t.blockNum+uint16(idx))
		} else {
			tftpBlock = createTFTPBlock(buffer[idx*t.blockSize:], t.blockNum+uint16(idx))
		}
		// We have to copy the block number here because the lamba function of the Do() is executed after the loop has finished.
		// Therefore the block number in the lambda function corresponds to the last value of the block number after the execution of the for-loop.
		ackNum := t.blockNum + uint16(idx)
		gomock.InOrder(
			t.fileTransferConnection.EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnection.EXPECT().
				ReadFromUDP(gomock.Any()).
				Do(func(b []byte) {
					binary.BigEndian.PutUint16(b, tftp.ACK.Value())
					binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], ackNum)
				}).Times(1),
		)
	}
	t.blockNum += uint16(end)

	if len(buffer)%t.blockSize == 0 {
		tftpBlock := createTFTPBlock(make([]byte, 0), t.blockNum)
		ackNum := t.blockNum
		gomock.InOrder(
			t.fileTransferConnection.EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnection.EXPECT().
				ReadFromUDP(gomock.Any()).
				Do(func(b []byte) {
					binary.BigEndian.PutUint16(b, tftp.ACK.Value())
					binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], ackNum)
				}).Times(1),
		)
	}

	t.blockNum++

	return t
}

func (t *TFTPFlowControl) createRetransmission(buffer []byte, times int, success bool) *TFTPFlowControl {
	tftpBlock := createTFTPBlock(buffer, t.blockNum)

	for try := 0; try < times; try++ {
		gomock.InOrder(
			t.fileTransferConnection.EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnection.EXPECT().
				ReadFromUDP(gomock.Any()).
				Do(func(b []byte) {
					binary.BigEndian.PutUint16(b, tftp.ACK.Value())
					binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], t.blockNum-uint16(1))
				}).Times(1),
		)

	}

	if success {
		gomock.InOrder(
			t.fileTransferConnection.EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnection.EXPECT().
				ReadFromUDP(gomock.Any()).
				Do(func(b []byte) {
					binary.BigEndian.PutUint16(b, tftp.ACK.Value())
					binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], t.blockNum)
				}).Times(1),
		)
	}
	return t
}

func (t *TFTPFlowControl) createOACK(blockSize int, fileSize int) *TFTPFlowControl {
	t.blockSize = blockSize

	oackbuffer := make([]byte, 2)
	binary.BigEndian.PutUint16(oackbuffer, tftp.OACK.Value())

	if blockSize != tftp.DefaultBlocksize {
		oackbuffer = append(oackbuffer, []byte("blksize")...)
		oackbuffer = append(oackbuffer, 0)
		oackbuffer = append(oackbuffer, []byte(strconv.Itoa(blockSize))...)
		oackbuffer = append(oackbuffer, 0)
	}
	if fileSize != tftp.DefaultTransfersize {
		oackbuffer = append(oackbuffer, []byte("tsize")...)
		oackbuffer = append(oackbuffer, 0)
		oackbuffer = append(oackbuffer, []byte(strconv.Itoa(fileSize))...)
		oackbuffer = append(oackbuffer, 0)
	}
	t.fileTransferConnection.EXPECT().Write(oackbuffer).Return(len(oackbuffer), nil).Times(1)
	t.fileTransferConnection.EXPECT().ReadFromUDP(gomock.Any()).Do(func(b []byte) {
		binary.BigEndian.PutUint16(b, tftp.ACK.Value())
		binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], uint16(0))
	}).Times(1)

	return t
}

func (t *TFTPFlowControl) createError(errorCode tftp.ErrorCode) *TFTPFlowControl {
	t.fileTransferConnection.EXPECT().Write(matchError(errorCode)).DoAndReturn(func(b []byte) (int, error) { return len(b), nil })
	return t
}

func (t *TFTPFlowControl) createWriteTimeout() *TFTPFlowControl {
	t.fileTransferConnection.EXPECT().Write(gomock.Any()).Return(0, errors.New("timeout")).Times(1)
	t.fileTransferConnection.EXPECT().ReadFromUDP(gomock.Any()).Times(0)
	return t
}

func (t *TFTPFlowControl) createReadTimeout() *TFTPFlowControl {
	t.fileTransferConnection.EXPECT().Write(gomock.Any()).DoAndReturn(func(b []byte) (int, error) { return len(b), nil })
	t.fileTransferConnection.EXPECT().ReadFromUDP(gomock.Any()).Return(0, nil, errors.New("timeout")).Times(3)
	return t
}

func createTFTPBlock(content []byte, blockNum uint16) []byte {
	var block = make([]byte, tftp.MetaInfoSize)
	binary.BigEndian.PutUint16(block, tftp.DATA.Value())
	binary.BigEndian.PutUint16(block[tftp.OpcodeLength:], blockNum)
	return append(block, content...)
}

func matchError(errorCode tftp.ErrorCode) gomock.Matcher {
	return &errorMatcher{
		errorCode: errorCode,
	}
}

type errorMatcher struct {
	errorCode tftp.ErrorCode
	want      []byte
}

func (m *errorMatcher) Matches(x interface{}) bool {
	expectedError := make([]byte, tftp.MetaInfoSize)
	binary.BigEndian.PutUint16(expectedError, tftp.ServerError.Value())
	binary.BigEndian.PutUint16(expectedError[tftp.OpcodeLength:], m.errorCode.Value())
	m.want = expectedError
	switch actualError := x.(type) {
	case []byte:
		return reflect.DeepEqual(actualError[:tftp.MetaInfoSize], expectedError)
	default:
		return false
	}
}

func (m *errorMatcher) String() string {
	return fmt.Sprintf("%v", m.want)
}
