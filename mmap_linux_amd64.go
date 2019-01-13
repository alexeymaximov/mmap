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

// Mapping.
type Mapping struct {
	internal
	alignedAddress uintptr
	alignedLength  uintptr
	locked         bool
}

// Make new mapping.
// Actual mapping offset and length may be different than specified
// by the reason of aligning to page size.
func New(fd uintptr, offset int64, length uintptr, mode Mode, flags Flag) (*Mapping, error) {

	// Using int64 (off_t) for offset and uintptr (size_t) for length by reason of compatibility.
	if offset < 0 {
		return nil, &ErrorInvalidOffset{Offset: offset}
	}
	if length > uintptr(maxInt) {
		return nil, &ErrorInvalidLength{Length: length}
	}

	mapping := &Mapping{}
	protection := syscall.PROT_READ
	mmapFlags := syscall.MAP_SHARED
	if mode < ModeReadOnly || mode > ModeWriteCopy {
		return nil, &ErrorInvalidMode{Mode: mode}
	}
	if mode > ModeReadOnly {
		protection |= syscall.PROT_WRITE
		mapping.writable = true
	}
	if mode == ModeWriteCopy {
		flags = syscall.MAP_PRIVATE
	}
	if flags&FlagExecutable != 0 {
		protection |= syscall.PROT_EXEC
		mapping.executable = true
	}

	// Mapping offset must be aligned by memory page size.
	pageSize := int64(os.Getpagesize())
	if pageSize < 0 {
		return nil, os.NewSyscallError("getpagesize", syscall.EINVAL)
	}
	outerOffset := offset / pageSize
	innerOffset := offset % pageSize
	mapping.alignedLength = uintptr(innerOffset) + length

	var err error
	mapping.alignedAddress, err = mmap(0, mapping.alignedLength, protection, mmapFlags, fd, outerOffset)
	if err != nil {
		return nil, os.NewSyscallError("mmap", err)
	}
	mapping.address = mapping.alignedAddress + uintptr(innerOffset)
	mapping.length = length

	// Convert mapping to byte slice at required offset.
	var sliceHeader struct {
		data uintptr
		len  int
		cap  int
	}
	sliceHeader.data = mapping.address
	sliceHeader.len = int(mapping.length)
	sliceHeader.cap = sliceHeader.len
	mapping.memory = *(*[]byte)(unsafe.Pointer(&sliceHeader))

	runtime.SetFinalizer(mapping, (*Mapping).Close)
	return mapping, nil
}

// Lock mapped memory pages.
// All pages that contain a part of mapping address range
// are guaranteed to be resident in RAM when the call returns successfully.
// The pages are guaranteed to stay in RAM until later unlocked.
// It may need to increase process memory limits for operation success.
// See working set on Windows and rlimit on Linux for details.
func (mapping *Mapping) Lock() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}
	if mapping.locked {
		return &ErrorLocked{}
	}
	if err := mlock(mapping.alignedAddress, mapping.alignedLength); err != nil {
		return os.NewSyscallError("mlock", err)
	}
	mapping.locked = true
	return nil
}

// Unlock mapped memory pages.
func (mapping *Mapping) Unlock() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}
	if !mapping.locked {
		return &ErrorUnlocked{}
	}
	if err := munlock(mapping.alignedAddress, mapping.alignedLength); err != nil {
		return os.NewSyscallError("munlock", err)
	}
	mapping.locked = false
	return nil
}

// Synchronize mapping with the underlying file.
func (mapping *Mapping) Sync() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}
	if !mapping.writable {
		return &ErrorIllegalOperation{Operation: "sync"}
	}
	return os.NewSyscallError("msync", msync(mapping.alignedAddress, mapping.alignedLength))
}

// Close mapping. Mapping will be synchronized with the underlying file and unlocked automatically.
// Implementation of io.Closer.
func (mapping *Mapping) Close() error {
	if mapping.memory == nil {
		return &ErrorClosed{}
	}

	// Maybe unnecessary.
	if mapping.writable {
		if err := mapping.Sync(); err != nil {
			return err
		}
	}
	if mapping.locked {
		if err := mapping.Unlock(); err != nil {
			return err
		}
	}

	if err := munmap(mapping.alignedAddress, mapping.alignedLength); err != nil {
		return os.NewSyscallError("munmap", err)
	}
	*mapping = Mapping{}
	runtime.SetFinalizer(mapping, nil)
	return nil
}
