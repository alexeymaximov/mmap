// Package mmap provides the cross-platform memory mapped file I/O.
// Note than all provided tools are not thread safe.
package mmap

import "io"

// Mode is a mapping mode.
type Mode int

const (
	// Share this mapping and allow read-only access.
	ModeReadOnly Mode = iota

	// Share this mapping.
	// Updates to the mapping are visible to other processes
	// mapping the same region, and are carried through to the underlying file.
	// To precisely control when updates are carried through to the underlying file
	// requires the use of Sync.
	ModeReadWrite

	// Create a private copy-on-write mapping.
	// Updates to the mapping are not visible to other processes
	// mapping the same region, and are not carried through to the underlying file.
	// It is unspecified whether changes made to the file are visible in the mapped region.
	ModeWriteCopy
)

// Flags is a mapping flags.
type Flag int

const (
	// Mapped memory pages may be executed.
	FlagExecutable Flag = 0x1
)

type internal struct {
	writable   bool
	executable bool
	address    uintptr
	memory     []byte
}

// Writable returns true if mapped memory pages may be written.
func (m *Mapping) Writable() bool {
	return m.writable
}

// Executable returns true if mapped memory pages may be executed.
func (m *Mapping) Executable() bool {
	return m.executable
}

// Address returns pointer to mapped memory.
func (m *Mapping) Address() uintptr {
	return m.address
}

// Length returns mapped memory length in bytes.
func (m *Mapping) Length() uintptr {
	return uintptr(len(m.memory))
}

// Memory returns mapped memory as a byte slice.
func (m *Mapping) Memory() []byte {
	return m.memory
}

// Begin starts the transaction for this mapping.
func (m *Mapping) Begin(offset int64, length uintptr) (*Transaction, error) {
	return NewTransaction(m, offset, length)
}

// Read reads len(buf) bytes at given offset from mapped memory.
// Implementation of io.ReaderAt.
func (m *Mapping) ReadAt(buf []byte, offset int64) (int, error) {
	if m.memory == nil {
		return 0, &ErrorClosed{}
	}
	if offset < 0 || offset >= int64(len(m.memory)) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(buf, m.memory[offset:])
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

// Write writes len(buf) bytes at given offset to mapped memory.
// Implementation of io.WriterAt.
func (m *Mapping) WriteAt(buf []byte, offset int64) (int, error) {
	if m.memory == nil {
		return 0, &ErrorClosed{}
	}
	if !m.writable {
		return 0, &ErrorIllegalOperation{Operation: "write"}
	}
	if offset < 0 || offset >= int64(len(m.memory)) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(m.memory[offset:], buf)
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}
