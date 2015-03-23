package tarfs

import (
	"archive/tar"
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// TODO: split out the creation of test tar file(s) so that this one
// giant test can be split into several.

func TestOpen(t *testing.T) {

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	// Add some files to the archive.
	var files = []struct {
		Name, Body string
	}{
		{"readme.txt", "This archive contains some text files."},
		{"gopher.txt", "Gopher names:\nGeorge\nGeoffrey\nGonzo\n"},
		{"todo.txt", "Get animal handling licence."},
		{"subdir/", ""},
		{"subdir/subreadme.txt", "An otherwise empty sub directory."},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Size: int64(len(file.Body)),
		}
		if strings.HasSuffix(file.Name, "/") {
			hdr.Typeflag = tar.TypeDir
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal("tar.WriteHeader:", err)
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			t.Fatal("tar.Write:", err)
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		t.Fatal("tar.Close:", err)
	}

	// Open the tar archive for reading.
	fs, err := New(buf)
	if err != nil {
		t.Fatal("New:", err)
	}

	for _, file := range files {
		f, err := fs.Open(file.Name)
		if err != nil {
			t.Errorf("Open(%q): %v", file.Name, err)
			continue
		}
		fi, err := f.Stat()
		if err != nil {
			t.Errorf("Stat of %q: %v", file.Name, err)
		}
		if strings.HasSuffix(file.Name, "/") != fi.IsDir() {
			t.Errorf("IsDir for %q: %v", file.Name, fi.IsDir())
		}
		content, err := ioutil.ReadAll(f)
		if err != nil {
			t.Errorf("ReadAll from %q: %v", file.Name, err)
			continue
		}
		if string(content) != file.Body {
			t.Errorf("For %q\nExpected:\n%q\nGot:\n%q\n", file.Name, file.Body, content)
		}
		if err = f.Close(); err != nil {
			t.Errorf("Close of %q: %v", file.Name, err)
		}
	}

	if _, err := fs.Open("foo\x00bar"); err == nil {
		t.Error("Open of filename containing \\x00 unexpectedly worked")
	}

	filename := "nosuchfile"
	if _, err := fs.Open(filename); err == nil {
		t.Errorf("Open(%q) unexpectedly worked", filename)
	} else if err != os.ErrNotExist {
		t.Errorf("Open(%q) gave %q, expected os.ErrNotExist", filename, err)
	}

	filename = "subdir/"
	d, err := fs.Open(filename)
	if err != nil {
		t.Fatalf("Open(%q): %v", err)
	}
	dir, err := d.Readdir(2)
	if err != nil {
		t.Fatal("Readdir on %q: %v:", err)
	}
	// TODO should this have "subdir/"? (it does)
	// TODO check results
	for _, fi := range dir {
		t.Log("Name:", fi.Name())
		t.Log("Size:", fi.Size())
		t.Log("Mode:", fi.Mode(), "IsDir:", fi.IsDir())
		t.Log("MTime:", fi.ModTime())
	}
	if len(dir) != 2 {
		t.Fatal("Unexpected Readdir results")
	}
}
