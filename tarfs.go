// Package tarfs is an in memory http.FileSystem from tars archives.
package tarfs

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// New returns an http.FileSystem that holds all the files in the tar,
// It reads the whole archive from the Reader.
// It is the caller's responsibility to call Close on the Reader when done.
func New(tarstream io.Reader) (http.FileSystem, error) {
	tr := tar.NewReader(tarstream)

	tarfs := make(tarfs)
	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(tr)
		if err != nil {
			return nil, err
		}

		tarfs[hdr.Name] = filedata{data: data, fi: hdr.FileInfo()}
	}
	return tarfs, nil
}

type filedata struct {
	data []byte
	fi   os.FileInfo
}

type file struct {
	*bytes.Reader
	fi    os.FileInfo
	files []os.FileInfo
}

type tarfs map[string]filedata

// Open implements http.FileSystem.
func (tf tarfs) Open(name string) (http.File, error) {
	if filepath.Separator != '/' && strings.IndexRune(name, filepath.Separator) >= 0 ||
		strings.Contains(name, "\x00") {
		return nil, errors.New("http: invalid character in file path")
	}
	fd, ok := tf[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	f := file{
		Reader: bytes.NewReader(fd.data),
		fi:     fd.fi,
	}
	if f.fi.IsDir() {
		f.files = make([]os.FileInfo, 0)
		for path, file := range tf {
			if strings.HasPrefix(path, name) {
				f.files = append(f.files, file.fi)
			}
		}

	}
	return &f, nil
}

// Close is a noop-closer.
func (f *file) Close() error {
	return nil
}

// Readdir implements http.File.
func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	// BUG(omeid): Does not implement the same semantics as
	// os.File.Readdir when n>0 && n<len(f.files).
	if f.fi.IsDir() && f.files != nil {
		return f.files, nil
	}
	return nil, os.ErrNotExist
}

// Stat implements http.File.
func (f *file) Stat() (os.FileInfo, error) {
	return f.fi, nil
}
