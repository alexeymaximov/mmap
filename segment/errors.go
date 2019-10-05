package segment

import "fmt"

// ErrorPartialRead is an error which returns when partial read occurred.
type ErrorPartialRead struct {
	// Index specifies the index of read value.
	Index int
	// Offset specifies the offset where this error occurred.
	Offset int64
	// NumBytes specifies the number of bytes were read.
	NumBytes int
}

// Implementation of the error interface.
func (err *ErrorPartialRead) Error() string {
	return fmt.Sprintf("segment: partial read of value #%d (%d bytes at 0x%x)", err.Index, err.NumBytes, err.Offset)
}

// ErrorPartialWrite is an error which returns when partial write occurred.
type ErrorPartialWrite struct {
	// Index specifies the index of written value.
	Index int
	// Offset specifies the offset where this error occurred.
	Offset int64
	// NumBytes specifies the number of bytes were written.
	NumBytes int
}

// Implementation of the error interface.
func (err *ErrorPartialWrite) Error() string {
	return fmt.Sprintf("segment: partial write of value #%d (%d bytes at 0x%x)", err.Index, err.NumBytes, err.Offset)
}

// ErrorUnsupportedType is an error which returns when the type of given value is unsupported.
type ErrorUnsupportedType struct {
	// Index specifies the index of unsupported value.
	Index int
}

// Implementation of the error interface.
func (err *ErrorUnsupportedType) Error() string {
	return fmt.Sprintf("segment: type of value #%d is not supported", err.Index)
}
