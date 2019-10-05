package mmap

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const maxInt = int(^uint(0) >> 1)

func errno(err error) error {
	if err != nil {
		if en, ok := err.(syscall.Errno); ok && en == 0 {
			return syscall.EINVAL
		}
		return err
	}
	return syscall.EINVAL
}

func mmap(addr, length uintptr, prot, flags int, fd uintptr, offset int64) (uintptr, error) {
	if prot < 0 || flags < 0 || offset < 0 {
		return 0, syscall.EINVAL
	}
	result, _, err := syscall.Syscall6(syscall.SYS_MMAP, addr, length, uintptr(prot), uintptr(flags), fd, uintptr(offset))
	if err != 0 {
		return 0, errno(err)
	}
	return result, nil
}

func mlock(addr, length uintptr) error {
	_, _, err := syscall.Syscall(syscall.SYS_MLOCK, addr, length, 0)
	if err != 0 {
		return errno(err)
	}
	return err
}

func munlock(addr, length uintptr) error {
	_, _, err := syscall.Syscall(syscall.SYS_MUNLOCK, addr, length, 0)
	if err != 0 {
		return errno(err)
	}
	return nil
}

func msync(addr, length uintptr) error {
	_, _, err := syscall.Syscall(syscall.SYS_MSYNC, addr, length, syscall.MS_SYNC)
	if err != 0 {
		return errno(err)
	}
	return nil
}

func munmap(addr, length uintptr) error {
	_, _, err := syscall.Syscall(syscall.SYS_MUNMAP, addr, length, 0)
	if err != 0 {
		return errno(err)
	}
	return nil
}

// Mapping is a mapping of the file into the memory.
type Mapping struct {
	internal
	alignedAddress uintptr
	alignedLength  uintptr
	locked         bool
}

// New returns a new mapping of the file into the memory.
// Actual offset and length may be different than the specified by the reason of aligning to page size.
func New(fd uintptr, offset int64, length uintptr, mode Mode, flags Flag) (*Mapping, error) {

	// Using int64 (off_t) for offset and uintptr (size_t) for the length by reason of compatibility.
	if offset < 0 {
		return nil, &ErrorInvalidOffset{Offset: offset}
	}
	if length > uintptr(maxInt) {
		return nil, &ErrorInvalidLength{Length: length}
	}

	m := &Mapping{}
	prot := syscall.PROT_READ
	mmapFlags := syscall.MAP_SHARED
	if mode < ModeReadOnly || mode > ModeWriteCopy {
		return nil, &ErrorInvalidMode{Mode: mode}
	}
	if mode > ModeReadOnly {
		prot |= syscall.PROT_WRITE
		m.writable = true
	}
	if mode == ModeWriteCopy {
		flags = syscall.MAP_PRIVATE
	}
	if flags&FlagExecutable != 0 {
		prot |= syscall.PROT_EXEC
		m.executable = true
	}

	// Mapping offset must be aligned by the memory page size.
	pageSize := int64(os.Getpagesize())
	if pageSize < 0 {
		return nil, os.NewSyscallError("getpagesize", syscall.EINVAL)
	}
	outerOffset := offset / pageSize
	innerOffset := offset % pageSize
	m.alignedLength = uintptr(innerOffset) + length

	var err error
	m.alignedAddress, err = mmap(0, m.alignedLength, prot, mmapFlags, fd, outerOffset)
	if err != nil {
		return nil, os.NewSyscallError("mmap", err)
	}
	m.address = m.alignedAddress + uintptr(innerOffset)

	// Convert the mapping into a byte slice.
	var sliceHeader struct {
		data uintptr
		len  int
		cap  int
	}
	sliceHeader.data = m.address
	sliceHeader.len = int(length)
	sliceHeader.cap = sliceHeader.len
	m.memory = *(*[]byte)(unsafe.Pointer(&sliceHeader))

	runtime.SetFinalizer(m, (*Mapping).Close)
	return m, nil
}

// Lock locks the mapped memory pages.
// All pages that contain a part of mapping address range
// are guaranteed to be resident in RAM when the call returns successfully.
// The pages are guaranteed to stay in RAM until later unlocked.
// It may need to increase process memory limits for operation success.
// See working set on Windows and rlimit on Linux for details.
func (m *Mapping) Lock() error {
	if m.memory == nil {
		return &ErrorClosed{}
	}
	if m.locked {
		return &ErrorLocked{}
	}
	if err := mlock(m.alignedAddress, m.alignedLength); err != nil {
		return os.NewSyscallError("mlock", err)
	}
	m.locked = true
	return nil
}

// Unlock unlocks the mapped memory pages.
func (m *Mapping) Unlock() error {
	if m.memory == nil {
		return &ErrorClosed{}
	}
	if !m.locked {
		return &ErrorUnlocked{}
	}
	if err := munlock(m.alignedAddress, m.alignedLength); err != nil {
		return os.NewSyscallError("munlock", err)
	}
	m.locked = false
	return nil
}

// Sync synchronizes this mapping with the underlying file.
func (m *Mapping) Sync() error {
	if m.memory == nil {
		return &ErrorClosed{}
	}
	if !m.writable {
		return &ErrorIllegalOperation{Operation: "sync"}
	}
	return os.NewSyscallError("msync", msync(m.alignedAddress, m.alignedLength))
}

// Close closes this mapping and frees all resources associated with it.
// Mapping will be synchronized with the underlying file and unlocked automatically.
// Implementation of io.Closer.
func (m *Mapping) Close() error {
	if m.memory == nil {
		return &ErrorClosed{}
	}

	// Maybe unnecessary.
	if m.writable {
		if err := m.Sync(); err != nil {
			return err
		}
	}
	if m.locked {
		if err := m.Unlock(); err != nil {
			return err
		}
	}

	if err := munmap(m.alignedAddress, m.alignedLength); err != nil {
		return os.NewSyscallError("munmap", err)
	}
	*m = Mapping{}
	runtime.SetFinalizer(m, nil)
	return nil
}
