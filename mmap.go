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
	size       uintptr
	data       []byte
	retained   []byte
}

// Check whether mapped memory pages may be written.
func (mapping *Mapping) Writable() bool {
	return mapping.writable
}

// Check whether mapped memory pages may be executed.
func (mapping *Mapping) Executable() bool {
	return mapping.executable
}

// Get pointer to mapping start.
func (mapping *Mapping) Address() uintptr {
	return mapping.address
}

// Get mapping size in bytes.
func (mapping *Mapping) Size() uintptr {
	return mapping.size
}

// Get mapped data.
func (mapping *Mapping) Data() []byte {
	return mapping.data
}

// Begin transaction.
// Mapped data will be copied to heap until commit or rollback.
func (mapping *Mapping) Begin() error {
	if mapping.data == nil {
		return &ErrorClosed{}
	}
	if mapping.retained != nil {
		return &ErrorTransactionStarted{}
	}
	if !mapping.writable {
		return &ErrorIllegalOperation{Operation: "transaction"}
	}
	snapshot := make([]byte, mapping.size)
	copy(snapshot, mapping.data)
	mapping.retained = mapping.data
	mapping.data = snapshot
	return nil
}

// Rollback transaction.
func (mapping *Mapping) Rollback() error {
	if mapping.data == nil {
		return &ErrorClosed{}
	}
	if mapping.retained == nil {
		return &ErrorTransactionNotStarted{}
	}
	mapping.data = mapping.retained
	mapping.retained = nil
	return nil
}

// Commit transaction.
func (mapping *Mapping) Commit() error {
	if mapping.data == nil {
		return &ErrorClosed{}
	}
	if mapping.retained == nil {
		return &ErrorTransactionNotStarted{}
	}
	copy(mapping.retained, mapping.data)
	mapping.data = mapping.retained
	mapping.retained = nil
	return nil
}

// Commit transaction if started and synchronize mapping with underlying file.
func (mapping *Mapping) Flush() error {
	if mapping.data == nil {
		return &ErrorClosed{}
	}
	if !mapping.writable {
		return &ErrorIllegalOperation{Operation: "flush"}
	}
	if mapping.retained != nil {
		if err := mapping.Commit(); err != nil {
			return err
		}
	}
	return mapping.Sync()
}

// Read len(buffer) bytes at given offset.
// Implementation of io.ReaderAt.
func (mapping *Mapping) ReadAt(buffer []byte, offset int64) (int, error) {
	if mapping.data == nil {
		return 0, &ErrorClosed{}
	}
	if offset < 0 || offset >= int64(mapping.size) {
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
	if !mapping.writable {
		return 0, &ErrorIllegalOperation{Operation: "write"}
	}
	if offset < 0 || offset >= int64(mapping.size) {
		return 0, &ErrorInvalidOffset{Offset: offset}
	}
	n := copy(mapping.data[offset:], buffer)
	if n < len(buffer) {
		return n, io.EOF
	}
	return n, nil
}
