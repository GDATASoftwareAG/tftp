package tftp_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"testing/fstest"

	"github.com/gdatasoftwareag/tftp/v2/pkg/handler"
	"github.com/gdatasoftwareag/tftp/v2/pkg/logging"
	"github.com/gdatasoftwareag/tftp/v2/pkg/tftp"
)

func Test_responseHandling_OpenFile(t *testing.T) {
	t.Parallel()
	testLogger := logging.CreateTestLogger(t, false)

	tests := []struct {
		name                       string
		files                      []string
		createResponseHandlingFunc func() tftp.ResponseHandling
		outMatcherFunc             func(io.ReadCloser, int64, error) error
	}{
		{
			name:  "Ensure handlers are checked in order of registration - HealthCheckHandler first",
			files: []string{"health"},
			createResponseHandlingFunc: func() tftp.ResponseHandling {
				responseHandling := tftp.NewResponseHandling()
				responseHandling.RegisterHandler(handler.NewHealthCheckHandler())
				responseHandling.RegisterHandler(handler.NewFSHandler(fstest.MapFS{
					"health": &fstest.MapFile{
						Data: []byte("FSHandlerHealth"),
					},
				}, testLogger))
				return responseHandling
			},
			outMatcherFunc: func(closer io.ReadCloser, i int64, err error) error {
				defer closer.Close()

				wantContent := []byte(handler.HealthCheckContent)
				gotContent, _ := io.ReadAll(closer)
				if !bytes.Equal(gotContent, wantContent) {
					return fmt.Errorf("unexpected files from server - got: %v, want: %v", gotContent, wantContent)
				}
				return nil
			},
		},
		{
			name:  "Handlers registered in wrong order - Unexpected response",
			files: []string{"health"},
			createResponseHandlingFunc: func() tftp.ResponseHandling {
				responseHandling := tftp.NewResponseHandling()
				responseHandling.RegisterHandler(handler.NewFSHandler(fstest.MapFS{
					"health": &fstest.MapFile{
						Data: []byte("FSHandlerHealth"),
					},
				}, testLogger))
				responseHandling.RegisterHandler(handler.NewHealthCheckHandler())
				return responseHandling
			},
			outMatcherFunc: func(closer io.ReadCloser, i int64, err error) error {
				defer closer.Close()

				doNotWantContent := []byte(handler.HealthCheckContent)
				gotContent, _ := io.ReadAll(closer)
				if bytes.Equal(gotContent, doNotWantContent) {
					return fmt.Errorf("received health check content - got: %s, do not want: %s", gotContent, doNotWantContent)
				}
				return nil
			},
		},
		{
			name:  "No handler registered - error",
			files: []string{"health"},
			createResponseHandlingFunc: func() tftp.ResponseHandling {
				responseHandling := tftp.NewResponseHandling()
				return responseHandling
			},
			outMatcherFunc: func(closer io.ReadCloser, size int64, err error) error {
				if closer != nil {
					return fmt.Errorf("did not expect reader - got: %T", closer)
				}
				if size != 0 {
					return fmt.Errorf("did not expect size - got: %d", size)
				}
				if !errors.Is(err, tftp.FileErrors.ErrNoHandler) {
					return fmt.Errorf("expected no handler error - got: %v, want: %v", err, tftp.FileErrors.ErrNoHandler)
				}

				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseHandling := tt.createResponseHandlingFunc()

			for _, file := range tt.files {
				if err := tt.outMatcherFunc(responseHandling.OpenFile(context.Background(), file)); err != nil {
					t.Errorf(err.Error())
				}
			}
		})
	}
}
