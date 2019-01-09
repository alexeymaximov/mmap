package mmap

import "io"

// Mapping access mode.
type Mode int

const (
	// Mapping is shared but only accessible for reading.
	ModeReadOnly Mode = iota

	// Mapping is shared: updates are visible to other processes mapping the same region,
	// and are carried through to the underlying file.
	// To precisely control when updates are carried through to the underlying file requires the use of Sync.
	ModeReadWrite

	// Mapping is in copy-on-write mode: updates are not visible to other processes mapping the same region,
	// and are not carried through to the underlying file.
	// It is unspecified whether changes made to the file are visible in the mapped region.
	ModeReadWritePrivate
)

// Mapping options.
type Options struct {
	Mode       Mode // Access mode.
	Executable bool // Allow execution.
}

// Get mapping length.
func (mapping *Mapping) Len() int {
	return len(mapping.data)
}

// Check whether mapping is available for reading (currently is always available).
func (mapping *Mapping) CanRead() bool {
	return true
}

// Check whether mapping is available for writing.
func (mapping *Mapping) CanWrite() bool {
	return mapping.canWrite
}

// Check whether mapping is available for execution.
func (mapping *Mapping) CanExecute() bool {
	return mapping.canExecute
}

// Get byte slice in [low, high) offset range.
func (mapping *Mapping) Slice(low, high int64) ([]byte, error) {
	if mapping.data == nil {
		return nil, &ErrorClosed{}
	}
	length := int64(len(mapping.data))
	if low < 0 || low >= length {
		return nil, &ErrorInvalidOffset{Offset: low}
	}
	if high < 1 || high > length {
		return nil, &ErrorInvalidOffset{Offset: high}
	}
	if low >= high {
		return nil, &ErrorInvalidOffsetRange{Low: low, High: high - 1}
	}
	return mapping.data[low:high], nil
}

// Read single byte at given offset.
func (mapping *Mapping) ReadByteAt(offset int64) (byte, error) {
	if mapping.data == nil {
		return 0, &ErrorClosed{}
	}
	if offset < 0 || offset >= int64(len(mapping.data)) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	return mapping.data[offset], nil
}

// Write single byte at given offset.
func (mapping *Mapping) WriteByteAt(byte byte, offset int64) error {
	if mapping.data == nil {
		return &ErrorClosed{}
	}
	if !mapping.canWrite {
		return &ErrorNotAllowed{Operation: "write"}
	}
	if offset < 0 || offset >= int64(len(mapping.data)) {
		return &ErrorInvalidOffset{Offset: offset}
	}
	mapping.data[offset] = byte
	return nil
}

// Read len(buffer) bytes at given offset.
// Implementation of io.ReaderAt.
func (mapping *Mapping) ReadAt(buffer []byte, offset int64) (int, error) {
	if mapping.data == nil {
		return 0, &ErrorClosed{}
	}
	if offset < 0 || offset >= int64(len(mapping.data)) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(buffer, mapping.data[offset:])
	if n < len(buffer) {
		return n, io.EOF
	}
	return n, nil
}

// Write len(buffer) bytes at given offset.
// Implementation of io.WriterAt.
func (mapping *Mapping) WriteAt(buffer []byte, offset int64) (int, error) {
	if mapping.data == nil {
		return 0, &ErrorClosed{}
	}
	if !mapping.canWrite {
		return 0, &ErrorNotAllowed{Operation: "write"}
	}
	if offset < 0 || offset >= int64(len(mapping.data)) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(mapping.data[offset:], buffer)
	if n < len(buffer) {
		return n, io.EOF
	}
	return n, nil
}
