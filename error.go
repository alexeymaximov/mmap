package mmap

import "fmt"

// Error occurred when mapping is closed.
type ErrorClosed struct{}

func (err *ErrorClosed) Error() string {
	return "mmap: mapping closed"
}

// Error occurred when operation is illegal.
type ErrorIllegalOperation struct{ Operation string }

func (err *ErrorIllegalOperation) Error() string {
	return fmt.Sprintf("mmap: illegal operation (%s)", err.Operation)
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

// Error occurred when length is invalid.
type ErrorInvalidLength struct{ Length uintptr }

func (err *ErrorInvalidLength) Error() string {
	return fmt.Sprintf("mmap: invalid length %d", err.Length)
}

// Error occurred when mapping memory pages are already locked.
type ErrorLocked struct{}

func (err *ErrorLocked) Error() string {
	return "mmap: mapping locked"
}

// Error occurred when transaction is not started.
type ErrorTransactionNotStarted struct{}

func (err *ErrorTransactionNotStarted) Error() string {
	return "mmap: transaction not started"
}

// Error occurred when transaction is already started.
type ErrorTransactionStarted struct{}

func (err *ErrorTransactionStarted) Error() string {
	return "mmap: transaction started"
}

// Error occurred when mapping memory pages are not locked.
type ErrorUnlocked struct{}

func (err *ErrorUnlocked) Error() string {
	return "mmap: mapping unlocked"
}
