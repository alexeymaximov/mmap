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

func testClose(t *testing.T, closer io.Closer) {
	if err := closer.Close(); err != nil {
		if _, ok := err.(*ErrorClosed); !ok {
			t.Fatal(err)
		}
	}
}

func makeTestFile(t *testing.T, rewrite bool) (*os.File, error) {
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
		testClose(t, file)
		return nil, err
	}
	return file, nil
}

func makeTestMapping(t *testing.T, mode Mode) (*Mapping, error) {
	file, err := makeTestFile(t, true)
	if err != nil {
		return nil, err
	}
	defer testClose(t, file)
	return New(file.Fd(), 0, testLength, mode, 0)
}

func TestOpenedFile(t *testing.T) {
	file, err := makeTestFile(t, true)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, file)
	mapping, err := New(file.Fd(), 0, testLength, ModeReadWrite, 0)
	defer testClose(t, mapping)
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
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

func TestClosedFile(t *testing.T) {
	mapping, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, mapping)
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
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
	mapping, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, mapping)
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := mapping.Sync(); err != nil {
		t.Fatal(err)
	}
	file, err := makeTestFile(t, false)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, file)
	buffer := make([]byte, len(testBuffer))
	if _, err := file.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buffer)
	}
}

func TestPrivateSync(t *testing.T) {
	mapping, err := makeTestMapping(t, ModeWriteCopy)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, mapping)
	if _, err := mapping.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := mapping.Sync(); err != nil {
		t.Fatal(err)
	}
	file, err := makeTestFile(t, false)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, file)
	buffer := make([]byte, len(testBuffer))
	if _, err := file.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, emptyBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", emptyBuffer, buffer)
	}
}

func TestPartialIO(t *testing.T) {
	file, err := makeTestFile(t, true)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, file)
	partialLength := uintptr(len(testBuffer) - 1)
	mapping, err := New(file.Fd(), 0, partialLength, ModeReadWrite, 0)
	defer testClose(t, mapping)
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
	file, err := makeTestFile(t, true)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, file)
	offLength := uintptr(len(testBuffer) - 1)
	mapping, err := New(file.Fd(), 1, offLength, ModeReadWrite, 0)
	defer testClose(t, mapping)
	offBuffer := make([]byte, offLength)
	copy(offBuffer, testBuffer[1:])
	if _, err := mapping.WriteAt(offBuffer, 0); err != nil {
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

func TestTransactionRollback(t *testing.T) {
	mapping, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, mapping)
	tx, err := mapping.Begin(0, mapping.Length())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, len(emptyBuffer))
	if _, err := mapping.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, emptyBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", emptyBuffer, buffer)
	}
}

func TestTransactionCommit(t *testing.T) {
	mapping, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, mapping)
	tx, err := mapping.Begin(0, mapping.Length())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.Flush(); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, len(testBuffer))
	if _, err := mapping.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buffer)
	}
	file, err := makeTestFile(t, false)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, file)
	buffer = make([]byte, len(testBuffer))
	if _, err := file.ReadAt(buffer, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buffer, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buffer)
	}
}
