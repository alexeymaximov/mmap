package mmap

import (
	"math"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const maxInt = int(^uint(0) >> 1)

// Mapping represents a mapping of file into the memory.
type Mapping struct {
	internal
	hProcess       syscall.Handle
	hFile          syscall.Handle
	hMapping       syscall.Handle
	alignedAddress uintptr
	alignedLength  uintptr
	locked         bool
}

// New returns a new mapping of file into the memory.
// Actual offset and length may be different than specified by the reason of aligning to page size.
func New(fd uintptr, offset int64, length uintptr, mode Mode, flags Flag) (*Mapping, error) {

	// Using int64 (off_t) for offset and uintptr (size_t) for length by reason of compatibility.
	if offset < 0 {
		return nil, &ErrorInvalidOffset{Offset: offset}
	}
	if length > uintptr(maxInt) {
		return nil, &ErrorInvalidLength{Length: length}
	}

	m := &Mapping{}
	prot := uint32(syscall.PAGE_READONLY)
	access := uint32(syscall.FILE_MAP_READ)
	switch mode {
	case ModeReadOnly:
		// NOOP
	case ModeReadWrite:
		prot = syscall.PAGE_READWRITE
		access = syscall.FILE_MAP_WRITE
		m.writable = true
	case ModeWriteCopy:
		prot = syscall.PAGE_WRITECOPY
		access = syscall.FILE_MAP_COPY
		m.writable = true
	default:
		return nil, &ErrorInvalidMode{Mode: mode}
	}
	if flags&FlagExecutable != 0 {
		prot <<= 4
		access |= syscall.FILE_MAP_EXECUTE
		m.executable = true
	}

	// Separate file handle needed to avoid errors on passed file external closing.
	var err error
	m.hProcess, err = syscall.GetCurrentProcess()
	if err != nil {
		return nil, os.NewSyscallError("GetCurrentProcess", err)
	}
	err = syscall.DuplicateHandle(
		m.hProcess, syscall.Handle(fd),
		m.hProcess, &m.hFile,
		0, true, syscall.DUPLICATE_SAME_ACCESS,
	)
	if err != nil {
		return nil, os.NewSyscallError("DuplicateHandle", err)
	}

	// Mapping offset must be aligned by memory page size.
	pageSize := int64(os.Getpagesize())
	if pageSize < 0 {
		return nil, os.NewSyscallError("getpagesize", syscall.EINVAL)
	}
	outerOffset := offset / pageSize
	innerOffset := offset % pageSize
	m.alignedLength = uintptr(innerOffset) + length

	maxSize := uint64(outerOffset) + uint64(m.alignedLength)
	maxSizeHigh := uint32(maxSize >> 32)
	maxSizeLow := uint32(maxSize & uint64(math.MaxUint32))
	m.hMapping, err = syscall.CreateFileMapping(m.hFile, nil, prot, maxSizeHigh, maxSizeLow, nil)
	if err != nil {
		return nil, os.NewSyscallError("CreateFileMapping", err)
	}
	fileOffset := uint64(outerOffset)
	fileOffsetHigh := uint32(fileOffset >> 32)
	fileOffsetLow := uint32(fileOffset & uint64(math.MaxUint32))
	m.alignedAddress, err = syscall.MapViewOfFile(
		m.hMapping, access,
		fileOffsetHigh, fileOffsetLow, m.alignedLength,
	)
	if err != nil {
		return nil, os.NewSyscallError("MapViewOfFile", err)
	}
	m.address = m.alignedAddress + uintptr(innerOffset)

	// Convert mapping to byte slice at required offset.
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

// Lock locks mapped memory pages.
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
	if err := syscall.VirtualLock(m.alignedAddress, m.alignedLength); err != nil {
		return os.NewSyscallError("VirtualLock", err)
	}
	m.locked = true
	return nil
}

// Unlock unlocks mapped memory pages.
func (m *Mapping) Unlock() error {
	if m.memory == nil {
		return &ErrorClosed{}
	}
	if !m.locked {
		return &ErrorUnlocked{}
	}
	if err := syscall.VirtualUnlock(m.alignedAddress, m.alignedLength); err != nil {
		return os.NewSyscallError("VirtualUnlock", err)
	}
	m.locked = false
	return nil
}

// Sync synchronizes mapping with the underlying file.
func (m *Mapping) Sync() error {
	if m.memory == nil {
		return &ErrorClosed{}
	}
	if !m.writable {
		return &ErrorIllegalOperation{Operation: "sync"}
	}
	if err := syscall.FlushViewOfFile(m.alignedAddress, m.alignedLength); err != nil {
		return os.NewSyscallError("FlushViewOfFile", err)
	}
	if err := syscall.FlushFileBuffers(m.hFile); err != nil {
		return os.NewSyscallError("FlushFileBuffers", err)
	}
	return nil
}

// Close closes this mapping and frees all resources associated with it.
// Mapping will be synchronized with the underlying file and unlocked automatically.
// Implementation of io.Closer.
func (m *Mapping) Close() error {
	if m.memory == nil {
		return &ErrorClosed{}
	}
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
	if err := syscall.UnmapViewOfFile(m.alignedAddress); err != nil {
		return os.NewSyscallError("UnmapViewOfFile", err)
	}
	if err := syscall.CloseHandle(m.hMapping); err != nil {
		return os.NewSyscallError("CloseHandle", err)
	}
	if err := syscall.CloseHandle(m.hFile); err != nil {
		return os.NewSyscallError("CloseHandle", err)
	}
	*m = Mapping{}
	runtime.SetFinalizer(m, nil)
	return nil
}
