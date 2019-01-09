package mmap

import "fmt"

type ErrorClosed struct{}

func (err *ErrorClosed) Error() string {
	return "mmap: mapping closed"
}

type ErrorInvalidMode struct{ Mode Mode }

func (err *ErrorInvalidMode) Error() string {
	return fmt.Sprintf("mmap: invalid mode 0x%x", err.Mode)
}

type ErrorInvalidOffset struct{ Offset int64 }

func (err *ErrorInvalidOffset) Error() string {
	return fmt.Sprintf("mmap: invalid offset 0x%x", err.Offset)
}

type ErrorInvalidOffsetRange struct{ Low, High int64 }

func (err *ErrorInvalidOffsetRange) Error() string {
	return fmt.Sprintf("mmap: invalid offset range 0x%x..0x%x", err.Low, err.High)
}

type ErrorInvalidSize struct{ Size uintptr }

func (err *ErrorInvalidSize) Error() string {
	return fmt.Sprintf("mmap: invalid size %d", err.Size)
}

type ErrorNotAllowed struct{ Operation string }

func (err *ErrorNotAllowed) Error() string {
	return fmt.Sprintf("mmap: %s is not allowed", err.Operation)
}
