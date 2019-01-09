package mmap

import "fmt"

// Error occurred when mapping is closed.
type ErrorClosed struct{}

func (err *ErrorClosed) Error() string {
	return "mmap: mapping closed"
}

// Error occurred when mapping mode is invalid.
type ErrorInvalidMode struct{ Mode Mode }

func (err *ErrorInvalidMode) Error() string {
	return fmt.Sprintf("mmap: invalid mode 0x%x", err.Mode)
}

// Error occurred when offset is invalid.
type ErrorInvalidOffset struct{ Offset int64 }

func (err *ErrorInvalidOffset) Error() string {
	return fmt.Sprintf("mmap: invalid offset 0x%x", err.Offset)
}

// Error occurred when offset range is invalid.
type ErrorInvalidOffsetRange struct{ Low, High int64 }

func (err *ErrorInvalidOffsetRange) Error() string {
	return fmt.Sprintf("mmap: invalid offset range 0x%x..0x%x", err.Low, err.High)
}

// Error occurred when mapping size is invalid.
type ErrorInvalidSize struct{ Size uintptr }

func (err *ErrorInvalidSize) Error() string {
	return fmt.Sprintf("mmap: invalid size %d", err.Size)
}

// Error occurred when operation on mapping is not allowed.
type ErrorNotAllowed struct{ Operation string }

func (err *ErrorNotAllowed) Error() string {
	return fmt.Sprintf("mmap: %s is not allowed", err.Operation)
}
