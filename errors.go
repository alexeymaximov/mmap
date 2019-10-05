package mmap

import "fmt"

// ErrorClosed is an error which returns when tries to access the closed mapping.
type ErrorClosed struct{}

// Implementation of the error interface.
func (err *ErrorClosed) Error() string {
	return "mmap: mapping closed"
}

// ErrorIllegalOperation is an error which returns when tries to execute illegal operation for the mapping.
type ErrorIllegalOperation struct{ Operation string }

// Implementation of the error interface.
func (err *ErrorIllegalOperation) Error() string {
	return fmt.Sprintf("mmap: illegal operation (%s)", err.Operation)
}

// ErrorInvalidLength is an error which returns when specified length is invalid.
type ErrorInvalidLength struct{ Length uintptr }

// Implementation of the error interface.
func (err *ErrorInvalidLength) Error() string {
	return fmt.Sprintf("mmap: invalid length %d", err.Length)
}

// ErrorInvalidMode is an error which returns when specified mapping mode is invalid.
type ErrorInvalidMode struct{ Mode Mode }

// Implementation of the error interface.
func (err *ErrorInvalidMode) Error() string {
	return fmt.Sprintf("mmap: invalid mode 0x%x", err.Mode)
}

// ErrorInvalidOffset is an error which returns when specified offset is invalid.
type ErrorInvalidOffset struct{ Offset int64 }

// Implementation of the error interface.
func (err *ErrorInvalidOffset) Error() string {
	return fmt.Sprintf("mmap: invalid offset 0x%x", err.Offset)
}

// ErrorLocked is an error which returns when mapping memory pages were already locked.
type ErrorLocked struct{}

// Implementation of the error interface.
func (err *ErrorLocked) Error() string {
	return "mmap: mapping locked"
}

// ErrorPartialCommit is an error which returns when the transaction was committed partially.
type ErrorPartialCommit struct{ BytesCommitted int }

// Implementation of the error interface.
func (err *ErrorPartialCommit) Error() string {
	return fmt.Sprintf("mmap: partial commit (%d bytes)", err.BytesCommitted)
}

// ErrorTransactionClosed is an error which returns when tries to access the closed transaction.
type ErrorTransactionClosed struct{}

// Implementation of the error interface.
func (err *ErrorTransactionClosed) Error() string {
	return fmt.Sprintf("mmap: transaction closed")
}

// ErrorUnlocked is an error which returns when mapping memory pages were not locked.
type ErrorUnlocked struct{}

// Implementation of the error interface.
func (err *ErrorUnlocked) Error() string {
	return "mmap: mapping unlocked"
}
