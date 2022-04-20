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
		name                     string
		fields                   fields
		expectedTransmittedFiles int
		wantErr                  bool
	}{
		{
			name: "Request file smaller than blocksize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, []string{"smallFile.txt"}).
						createTFTPBlocksFromBuffer(readFileToBuffer("smallFile.txt"), 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  false,
		},
		{
			name: "Request file equals blocksize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, []string{"defaultBlockSizeFile.txt"}).
						createTFTPBlocksFromBuffer(readFileToBuffer("defaultBlockSizeFile.txt"), 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  false,
		},
		{
			name: "Request file larger blocksize",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, []string{"largeFile.txt"}).
						createTFTPBlocksFromBuffer(readFileToBuffer("largeFile.txt"), 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  false,
		},
		{
			name: "Send file twice",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					files := []string{"smallFile.txt", "largeFile.txt"}
					controller := setupFileTransferConnections(ctrl, files).
						createTFTPBlocksFromFiles(files)

					return controller
				},
			},
			expectedTransmittedFiles: 2,
			wantErr:                  false,
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
					controller := setupFileTransferConnections(ctrl, []string{"defaultBlockSizeFile.txt"}, options...).
						createOACK(123, len(fileBuffer), 0).
						createTFTPBlocksFromBuffer(fileBuffer, 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  false,
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
					controller := setupFileTransferConnections(ctrl, []string{"defaultBlockSizeFile.txt"}, options...).
						createOACK(123, tftp.DefaultTransfersize, 0).
						createTFTPBlocksFromBuffer(fileBuffer, 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  false,
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
					controller := setupFileTransferConnections(ctrl, []string{"defaultBlockSizeFile.txt"}, options...).
						createOACK(tftp.DefaultBlocksize, len(fileBuffer), 0).
						createTFTPBlocksFromBuffer(fileBuffer, 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  false,
		},
		{
			name: "Test file not found",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, []string{"not_present_file.txt"}).
						createError(tftp.NotFound, 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  true,
		},
		{
			name: "Test read timeout",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, []string{"defaultBlockSizeFile.txt"}).
						createReadTimeout(0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  true,
		},
		{
			name: "Test write timeout",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					controller := setupFileTransferConnections(ctrl, []string{"smallFile.txt"}).
						createWriteTimeout(0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  true,
		},
		{
			name: "Test retransmission of package",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					fileBuffer := readFileToBuffer("smallFile.txt")
					controller := setupFileTransferConnections(ctrl, []string{"smallFile.txt"}).
						createRetransmission(fileBuffer, 1, true, 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  false,
		},
		{
			name: "Test retransmission failure of package",
			fields: fields{
				tftpFlowControlFunc: func(ctrl *gomock.Controller) *TFTPFlowControl {
					fileBuffer := readFileToBuffer("smallFile.txt")
					controller := setupFileTransferConnections(ctrl, []string{"smallFile.txt"}).
						createRetransmission(fileBuffer, 3, false, 0)

					return controller
				},
			},
			expectedTransmittedFiles: 1,
			wantErr:                  true,
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
				MaxParallelConnections: 1,
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
			transmissionCounter := 0
			for transmissionCounter < tt.expectedTransmittedFiles {
				<-flowControl.transmissionCount
				transmissionCounter++
			}
			close(flowControl.transmissionCount)
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
	connector               *mock.MockConnector
	fileTransferConnections []*mock.MockConnection
	blockSize               int
	blockNum                uint16
	transmissionCount       chan struct{}
}

func createUDPConnectionWithIncomingRequests(ctrl *gomock.Controller, files []string, requestOptions ...tftp.Option) udp.Connection {
	listenForRRQConnection := mock.NewMockConnection(ctrl)

	listenForRRQConnection.
		EXPECT().
		ReadFromUDP(gomock.Any()).
		DoAndReturn(func(b []byte) (_ int, _ *net.UDPAddr, _ error) {
			if len(files) == 0 {
				return 0, clientAddress, io.EOF
			}
			file := files[0]
			files = files[1:]
			rrqData := tftp.CreateRRQTestData(file, tftp.ModeOctet, requestOptions)
			copy(b, rrqData)
			return len(rrqData), clientAddress, nil
		}).AnyTimes()
	listenForRRQConnection.EXPECT().Close()

	return listenForRRQConnection
}

func createFileTransferConnection(ctrl *gomock.Controller, transmissionCountChan chan struct{}) *mock.MockConnection {
	mockConnection := mock.NewMockConnection(ctrl)
	mockConnection.EXPECT().Close().Do(func() error {
		transmissionCountChan <- struct{}{}
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

func setupFileTransferConnections(ctrl *gomock.Controller, files []string, requestOptions ...tftp.Option) *TFTPFlowControl {
	mockConnector := mock.NewMockConnector(ctrl)
	mockConnector.
		EXPECT().
		ListenUDP("udp", gomock.Any()).
		Return(createUDPConnectionWithIncomingRequests(ctrl, files, requestOptions...), nil)

	// Server establishes connection to incoming request
	//    Get a local address with random high port
	mockConnector.EXPECT().
		ResolveUDPAddr("udp", listenAddress.IP.String()+":0").
		Return(resolvedListenAddress, nil).Times(len(files))

	transmissionCountChan := make(chan struct{})
	fileTransferConnections := make([]*mock.MockConnection, 0, len(files))
	for connectionIdx := 0; connectionIdx < len(files); connectionIdx++ {
		fileTransferConnections = append(fileTransferConnections, createFileTransferConnection(ctrl, transmissionCountChan))
	}

	mockConnector.
		EXPECT().
		DialUDP("udp", resolvedListenAddress, clientAddress).
		DoAndReturn(func(network string, laddr, raddr *net.UDPAddr) (udp.Connection, error) {
			fileTransferConnection := fileTransferConnections[0]
			fileTransferConnections = fileTransferConnections[1:]
			return fileTransferConnection, nil
		}).Times(len(files))
	// Establish connection for file transfer

	return &TFTPFlowControl{
		connector:               mockConnector,
		fileTransferConnections: fileTransferConnections,
		blockSize:               tftp.DefaultBlocksize,
		blockNum:                1,
		transmissionCount:       transmissionCountChan,
	}
}

func (t *TFTPFlowControl) createTFTPBlocksFromFiles(files []string) *TFTPFlowControl {
	for idx, file := range files {
		t.createTFTPBlocksFromBuffer(readFileToBuffer(file), idx)
	}
	return t
}

func (t *TFTPFlowControl) createTFTPBlocksFromBuffer(buffer []byte, connectionIdx int) *TFTPFlowControl {
	t.blockNum = 1
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
			t.fileTransferConnections[connectionIdx].EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnections[connectionIdx].EXPECT().
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
			t.fileTransferConnections[connectionIdx].EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnections[connectionIdx].EXPECT().
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

func (t *TFTPFlowControl) createRetransmission(buffer []byte, times int, success bool, connectionIdx int) *TFTPFlowControl {
	tftpBlock := createTFTPBlock(buffer, t.blockNum)

	for try := 0; try < times; try++ {
		gomock.InOrder(
			t.fileTransferConnections[connectionIdx].EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnections[connectionIdx].EXPECT().
				ReadFromUDP(gomock.Any()).
				Do(func(b []byte) {
					binary.BigEndian.PutUint16(b, tftp.ACK.Value())
					binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], t.blockNum-uint16(1))
				}).Times(1),
		)

	}

	if success {
		gomock.InOrder(
			t.fileTransferConnections[connectionIdx].EXPECT().Write(tftpBlock).Return(len(tftpBlock), nil).Times(1),
			t.fileTransferConnections[connectionIdx].EXPECT().
				ReadFromUDP(gomock.Any()).
				Do(func(b []byte) {
					binary.BigEndian.PutUint16(b, tftp.ACK.Value())
					binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], t.blockNum)
				}).Times(1),
		)
	}
	return t
}

func (t *TFTPFlowControl) createOACK(blockSize int, fileSize int, connectionIdx int) *TFTPFlowControl {
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
	t.fileTransferConnections[connectionIdx].EXPECT().Write(oackbuffer).Return(len(oackbuffer), nil).Times(1)
	t.fileTransferConnections[connectionIdx].EXPECT().ReadFromUDP(gomock.Any()).Do(func(b []byte) {
		binary.BigEndian.PutUint16(b, tftp.ACK.Value())
		binary.BigEndian.PutUint16(b[tftp.OpcodeLength:], uint16(0))
	}).Times(1)

	return t
}

func (t *TFTPFlowControl) createError(errorCode tftp.ErrorCode, connectionIdx int) *TFTPFlowControl {
	t.fileTransferConnections[connectionIdx].EXPECT().Write(matchError(errorCode)).DoAndReturn(func(b []byte) (int, error) { return len(b), nil })
	return t
}

func (t *TFTPFlowControl) createWriteTimeout(connectionIdx int) *TFTPFlowControl {
	t.fileTransferConnections[connectionIdx].EXPECT().Write(gomock.Any()).Return(0, errors.New("timeout")).Times(1)
	t.fileTransferConnections[connectionIdx].EXPECT().ReadFromUDP(gomock.Any()).Times(0)
	return t
}

func (t *TFTPFlowControl) createReadTimeout(connectionIdx int) *TFTPFlowControl {
	t.fileTransferConnections[connectionIdx].EXPECT().Write(gomock.Any()).DoAndReturn(func(b []byte) (int, error) { return len(b), nil }).Times(3)
	t.fileTransferConnections[connectionIdx].EXPECT().ReadFromUDP(gomock.Any()).Return(0, nil, errors.New("timeout")).Times(3)
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
