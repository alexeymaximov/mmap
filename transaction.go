package mmap

import (
	"io"
	"runtime"
)

// Transaction is a memory mapping transaction.
// The transaction is not valid if the parent mapping is closed.
type Transaction struct {
	mapping    *Mapping
	offset     int64
	highOffset int64
	snapshot   []byte
}

// Begin starts a transaction.
// Mapped memory starting from given offset and ends after given length
// copies to the transaction snapshot which is allocated into the heap.
func (m *Mapping) Begin(offset int64, length uintptr) (*Transaction, error) {
	if m.memory == nil {
		return nil, &ErrorClosed{}
	}
	if !m.writable {
		return nil, &ErrorIllegalOperation{Operation: "transaction"}
	}
	if offset < 0 || offset >= int64(len(m.memory)) {
		return nil, &ErrorInvalidOffset{Offset: offset}
	}
	highOffset := offset + int64(length)
	if length == 0 || highOffset > int64(len(m.memory)) {
		return nil, &ErrorInvalidLength{Length: length}
	}
	tx := &Transaction{
		mapping:    m,
		offset:     offset,
		highOffset: highOffset,
		snapshot:   make([]byte, length),
	}
	copy(tx.snapshot, m.memory[offset:highOffset])
	runtime.SetFinalizer(tx, (*Transaction).Rollback)
	return tx, nil
}

// Offset returns the starting offset of this transaction.
func (tx *Transaction) Offset() int64 {
	return tx.offset
}

// Length returns the snapshot length in bytes.
func (tx *Transaction) Length() uintptr {
	return uintptr(len(tx.snapshot))
}

// Read reads len(buf) bytes at given offset relatively to the parent mapping address from the snapshot.
// Implementation of io.ReaderAt.
func (tx *Transaction) ReadAt(buf []byte, offset int64) (int, error) {
	if tx.snapshot == nil {
		return 0, &ErrorTransactionClosed{}
	}
	if offset < tx.offset || offset >= tx.highOffset {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(buf, tx.snapshot[offset-tx.offset:])
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

// Write writes len(buf) bytes at given offset relatively to the parent mapping address to the snapshot.
// Implementation of io.WriterAt.
func (tx *Transaction) WriteAt(buf []byte, offset int64) (int, error) {
	if tx.snapshot == nil {
		return 0, &ErrorTransactionClosed{}
	}
	if offset < 0 || offset >= tx.highOffset {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(tx.snapshot[offset-tx.offset:], buf)
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

// Commit flushes the snapshot to the mapped memory, closes this transaction and frees all resources associated with it.
func (tx *Transaction) Commit() error {
	if tx.snapshot == nil {
		return &ErrorTransactionClosed{}
	}
	if tx.mapping.memory == nil {
		return &ErrorClosed{}
	}
	if n := copy(tx.mapping.memory[tx.offset:tx.highOffset], tx.snapshot); n < len(tx.snapshot) {
		return &ErrorPartialCommit{NumBytes: n}
	}
	tx.snapshot = nil
	return nil
}

// Rollback closes this transaction and frees all resources associated with it.
func (tx *Transaction) Rollback() error {
	if tx.snapshot == nil {
		return &ErrorTransactionClosed{}
	}
	tx.snapshot = nil
	return nil
}
