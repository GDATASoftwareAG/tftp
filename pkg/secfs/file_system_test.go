package secfs

import (
	"bytes"
	"io"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"
)

const (
	defaultRootPath = "/var/lib/tftpboot"
)

func Test_fileSystem_Open(t *testing.T) {
	type fields struct {
		rootPath string
		fs       fs.FS
	}
	type args struct {
		name string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantContent []byte
		wantErr     bool
	}{
		{
			name: "Open file - success",
			fields: fields{
				rootPath: defaultRootPath,
				fs: fstest.MapFS{
					"pxelinux.0": &fstest.MapFile{
						Data: []byte("Hello World"),
					},
				},
			},
			args: args{
				name: "pxelinux.0",
			},
			wantContent: []byte("Hello World"),
		},
		{
			name: "Open with leading slash - success",
			fields: fields{
				rootPath: defaultRootPath,
				fs: fstest.MapFS{
					"pxelinux.0": &fstest.MapFile{
						Data: []byte("Hello World"),
					},
				},
			},
			args: args{
				name: "/pxelinux.0",
			},
			wantContent: []byte("Hello World"),
		},
		{
			name: "Open file - not found",
			fields: fields{
				rootPath: defaultRootPath,
				fs:       fstest.MapFS{},
			},
			args: args{
				name: "pxelinux.0",
			},
			wantErr: true,
		},
		{
			name: "Open file from absolute path - error",
			fields: fields{
				rootPath: defaultRootPath,
				fs: fstest.MapFS{
					"pxelinux.0": &fstest.MapFile{
						Data: []byte("Hello World"),
					},
				},
			},
			args: args{
				name: filepath.Join(defaultRootPath, "pxelinux.0"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fileSystem{
				rootPath: tt.fields.rootPath,
				fs:       tt.fields.fs,
			}
			got, err := f.Open(tt.args.name)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("fileSystem.Open() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			var gotContent []byte
			if gotContent, err = io.ReadAll(got); err != nil {
				t.Fatalf("Failed to read file - error = %v", err)
			}
			if !bytes.Equal(gotContent, tt.wantContent) {
				t.Errorf("Content did not match = got: %v, want: %v", gotContent, tt.wantContent)
			}
		})
	}
}
