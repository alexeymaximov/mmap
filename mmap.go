package mmap

import "io"

// Mapping mode.
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

// Mapping flags.
type Flag int

const (
	// Mapped memory pages may be executed.
	FlagExecutable Flag = 0x1
)

type internal struct {
	writable   bool
	executable bool
	address    uintptr
	length     uintptr
	memory     []byte
	backup     []byte
}

// Check whether mapped memory pages may be written.
func (mapping *Mapping) Writable() bool {
	return mapping.writable
}

// Check whether mapped memory pages may be executed.
func (mapping *Mapping) Executable() bool {
	return mapping.executable
}

// Get pointer to mapped memory.
func (mapping *Mapping) Address() uintptr {
	return mapping.address
}

// Get mapped memory length in bytes.
func (mapping *Mapping) Length() uintptr {
	return mapping.length
}

// Get mapped memory as byte slice.
func (mapping *Mapping) Memory() []byte {
	return mapping.memory
}

// Begin transaction.
// Mapped memory will be copied into the heap until commit or rollback.
func (mapping *Mapping) Begin() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}
	if mapping.backup != nil {
		return &ErrorTransactionStarted{}
	}
	if !mapping.writable {
		return &ErrorIllegalOperation{Operation: "transaction"}
	}
	snapshot := make([]byte, mapping.length)
	copy(snapshot, mapping.memory)
	mapping.backup = mapping.memory
	mapping.memory = snapshot
	return nil
}

// Rollback transaction.
func (mapping *Mapping) Rollback() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}
	if mapping.backup == nil {
		return &ErrorTransactionNotStarted{}
	}
	mapping.memory = mapping.backup
	mapping.backup = nil
	return nil
}

// Commit transaction.
func (mapping *Mapping) Commit() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}
	if mapping.backup == nil {
		return &ErrorTransactionNotStarted{}
	}
	copy(mapping.backup, mapping.memory)
	mapping.memory = mapping.backup
	mapping.backup = nil
	return nil
}

// Commit transaction if started and synchronize mapping with the underlying file.
func (mapping *Mapping) Flush() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}
	if !mapping.writable {
		return &ErrorIllegalOperation{Operation: "flush"}
	}
	if mapping.backup != nil {
		if err := mapping.Commit(); err != nil {
			return err
		}
	}
	return mapping.Sync()
}

// Read len(buffer) bytes at given offset.
// Implementation of io.ReaderAt.
func (mapping *Mapping) ReadAt(buffer []byte, offset int64) (int, error) {
	if mapping.memory == nil {
		return 0, &ErrorClosed{}
	}
	if offset < 0 || offset >= int64(mapping.length) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(buffer, mapping.memory[offset:])
	if n < len(buffer) {
		return n, io.EOF
	}
	return n, nil
}

// Write len(buffer) bytes at given offset.
// Implementation of io.WriterAt.
func (mapping *Mapping) WriteAt(buffer []byte, offset int64) (int, error) {
	if mapping.memory == nil {
		return 0, &ErrorClosed{}
	}
	if !mapping.writable {
		return 0, &ErrorIllegalOperation{Operation: "write"}
	}
	if offset < 0 || offset >= int64(mapping.length) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(mapping.memory[offset:], buffer)
	if n < len(buffer) {
		return n, io.EOF
	}
	return n, nil
}
