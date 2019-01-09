package mmap

import "io"

type Mode int

const (
	ModeReadOnly Mode = iota
	ModeReadWrite
	ModeReadWritePrivate
)

type Options struct {
	Mode       Mode
	Executable bool
}

func (mapping *Mapping) Len() int {
	return len(mapping.data)
}

func (mapping *Mapping) CanRead() bool {
	return true
}

func (mapping *Mapping) CanWrite() bool {
	return mapping.canWrite
}

func (mapping *Mapping) CanExecute() bool {
	return mapping.canExecute
}

func (mapping *Mapping) Direct(low, high int64) ([]byte, error) {
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

func (mapping *Mapping) ReadByteAt(offset int64) (byte, error) {
	if mapping.data == nil {
		return 0, &ErrorClosed{}
	}
	if offset < 0 || offset >= int64(len(mapping.data)) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	return mapping.data[offset], nil
}

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
