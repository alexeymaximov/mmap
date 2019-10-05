package mmap

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

var testPath = filepath.Join(os.TempDir(), "mmap.test")
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
	f, err := os.OpenFile(testPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := f.Truncate(int64(testLength)); err != nil {
		testClose(t, f)
		return nil, err
	}
	return f, nil
}

func makeTestMapping(t *testing.T, mode Mode) (*Mapping, error) {
	f, err := makeTestFile(t, true)
	if err != nil {
		return nil, err
	}
	defer testClose(t, f)
	return New(f.Fd(), 0, testLength, mode, 0)
}

func TestOpenedFile(t *testing.T) {
	f, err := makeTestFile(t, true)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, f)
	m, err := New(f.Fd(), 0, testLength, ModeReadWrite, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	if _, err := m.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(testBuffer))
	if _, err := m.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buf)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClosedFile(t *testing.T) {
	m, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	if _, err := m.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(testBuffer))
	if _, err := m.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buf)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSharedSync(t *testing.T) {
	m, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	if _, err := m.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.Sync(); err != nil {
		t.Fatal(err)
	}
	f, err := makeTestFile(t, false)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, f)
	buf := make([]byte, len(testBuffer))
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buf)
	}
}

func TestPrivateSync(t *testing.T) {
	m, err := makeTestMapping(t, ModeWriteCopy)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	if _, err := m.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.Sync(); err != nil {
		t.Fatal(err)
	}
	f, err := makeTestFile(t, false)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, f)
	buf := make([]byte, len(testBuffer))
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, emptyBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", emptyBuffer, buf)
	}
}

func TestPartialIO(t *testing.T) {
	f, err := makeTestFile(t, true)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, f)
	partLen := uintptr(len(testBuffer) - 1)
	m, err := New(f.Fd(), 0, partLen, ModeReadWrite, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	if _, err := m.WriteAt(testBuffer, 0); err == nil {
		t.Fatal("expected io.EOF, no error found")
	} else if err != io.EOF {
		t.Fatalf("expected io.EOF, [%v] error found", err)
	}
	partBuf := make([]byte, len(testBuffer))
	copy(partBuf[0:partLen], testBuffer)
	buf := make([]byte, len(testBuffer))
	if _, err := m.ReadAt(buf, 0); err == nil {
		t.Fatal("expected io.EOF, no error found")
	} else if err != io.EOF {
		t.Fatalf("expected io.EOF, [%v] error found", err)
	}
	if bytes.Compare(buf, partBuf) != 0 {
		t.Fatalf("buffer must be a %v, %v found", partBuf, buf)
	}
}

func TestOffset(t *testing.T) {
	f, err := makeTestFile(t, true)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, f)
	offLen := uintptr(len(testBuffer) - 1)
	m, err := New(f.Fd(), 1, offLen, ModeReadWrite, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	offBuf := make([]byte, offLen)
	copy(offBuf, testBuffer[1:])
	if _, err := m.WriteAt(offBuf, 0); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(offBuf))
	if _, err := m.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, offBuf) != 0 {
		t.Fatalf("buffer must be a %q, %v found", offBuf, buf)
	}
}

func TestTransactionRollback(t *testing.T) {
	m, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	tx, err := m.Begin(0, m.Length())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(emptyBuffer))
	if _, err := m.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, emptyBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", emptyBuffer, buf)
	}
}

func TestTransactionCommit(t *testing.T) {
	m, err := makeTestMapping(t, ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, m)
	tx, err := m.Begin(0, m.Length())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.WriteAt(testBuffer, 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := m.Sync(); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(testBuffer))
	if _, err := m.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buf)
	}
	f, err := makeTestFile(t, false)
	if err != nil {
		t.Fatal(err)
	}
	defer testClose(t, f)
	buf = make([]byte, len(testBuffer))
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf, testBuffer) != 0 {
		t.Fatalf("buffer must be a %q, %v found", testBuffer, buf)
	}
}
