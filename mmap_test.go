package mmap

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

var testPath = filepath.Join(os.TempDir(), "test.mmap")
var testLength = uintptr(1 << 20)

var testBuffer = []byte{'H', 'E', 'L', 'L', 'O'}
var emptyBuffer = []byte{0, 0, 0, 0, 0}

func makeTestFile(rewrite bool) (*os.File, error) {
	if rewrite {
		if _, err := os.Stat(testPath); err == nil || !os.IsNotExist(err) {
			if err := os.Remove(testPath); err != nil {
				return nil, err
			}
		}
	}
	file, err := os.OpenFile(testPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := file.Truncate(int64(testLength)); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func makeTestMapping(mode Mode) (*Mapping, error) {
	file, err := makeTestFile(true)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return New(file.Fd(), 0, testLength, &Options{
		Mode: mode,
	})
}

func TestOpenedFile(t *testing.T) {
	file, err := makeTestFile(true)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	mapping, err := New(file.Fd(), 0, testLength, &Options{
		Mode: ModeReadWrite,
	})
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
		mapping.Close()
		t.Fatal(err)
	}
	defer mapping.Close()
	buffer := make([]byte, len(testBuffer))
	if _, err := mapping.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buffer)
	}
	if err := mapping.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClosedFile(t *testing.T) {
	mapping, err := makeTestMapping(ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer mapping.Close()
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
		mapping.Close()
		t.Fatal(err)
	}
	buffer := make([]byte, len(testBuffer))
	if _, err := mapping.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buffer)
	}
	if err := mapping.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSharedSync(t *testing.T) {
	mapping, err := makeTestMapping(ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer mapping.Close()
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := mapping.Sync(); err != nil {
		t.Fatal(err)
	}
	file, err := makeTestFile(false)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	buffer := make([]byte, len(testBuffer))
	if _, err := file.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buffer)
	}
}

func TestPrivateSync(t *testing.T) {
	mapping, err := makeTestMapping(ModeReadWritePrivate)
	if err != nil {
		t.Fatal(err)
	}
	defer mapping.Close()
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := mapping.Sync(); err != nil {
		t.Fatal(err)
	}
	file, err := makeTestFile(false)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	buffer := make([]byte, len(testBuffer))
	if _, err := file.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, emptyBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", emptyBuffer, buffer)
	}
}

func TestPartialIO(t *testing.T) {
	file, err := makeTestFile(true)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	partialLength := uintptr(len(testBuffer) - 1)
	mapping, err := New(file.Fd(), 0, partialLength, &Options{
		Mode: ModeReadWrite,
	})
	defer mapping.Close()
	if _, err := mapping.WriteAt(testBuffer, 0); err == nil {
		t.Fatal("expected io.EOF, no error found")
	} else if err != io.EOF {
		t.Fatalf("expected io.EOF, [%v] error found", err)
	}
	partialBuffer := make([]byte, len(testBuffer))
	copy(partialBuffer[0:partialLength], testBuffer)
	buffer := make([]byte, len(testBuffer))
	if _, err := mapping.ReadAt(buffer, 0); err == nil {
		t.Fatal("expected io.EOF, no error found")
	} else if err != io.EOF {
		t.Fatalf("expected io.EOF, [%v] error found", err)
	}
	if bytes.Compare(buffer, partialBuffer) != 0 {
		t.Fatalf("buffer must be a %v, %v found", partialBuffer, buffer)
	}
}

func TestOffset(t *testing.T) {
	file, err := makeTestFile(true)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	offLength := uintptr(len(testBuffer) - 1)
	mapping, err := New(file.Fd(), 1, offLength, &Options{
		Mode: ModeReadWrite,
	})
	defer mapping.Close()
	offBuffer := make([]byte, offLength)
	copy(offBuffer, testBuffer[1:])
	if _, err := mapping.WriteAt(offBuffer, 0); err != nil {
		mapping.Close()
		t.Fatal(err)
	}
	buffer := make([]byte, len(offBuffer))
	if _, err := mapping.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, offBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", offBuffer, buffer)
	}
}
