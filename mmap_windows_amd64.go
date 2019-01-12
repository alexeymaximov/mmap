package mmap

import (
	"math"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const maxInt = int(^uint(0) >> 1)

// Mapping.
type Mapping struct {
	internal
	hProcess       syscall.Handle
	hFile          syscall.Handle
	hMapping       syscall.Handle
	alignedAddress uintptr
	alignedSize    uintptr
}

// Make new mapping of file at unaligned offset.
func New(fd uintptr, offset int64, size uintptr, mode Mode, flags Flag) (*Mapping, error) {

	// Using int64 (off_t) for offset and uintptr (size_t) for size by reason of compatibility.
	if offset < 0 {
		return nil, &ErrorInvalidOffset{Offset: offset}
	}
	if size > uintptr(maxInt) {
		return nil, &ErrorInvalidSize{Size: size}
	}

	mapping := &Mapping{}
	protection := uint32(syscall.PAGE_READONLY)
	access := uint32(syscall.FILE_MAP_READ)
	switch mode {
	case ModeReadOnly:
		// NOOP
	case ModeReadWrite:
		protection = syscall.PAGE_READWRITE
		access = syscall.FILE_MAP_WRITE
		mapping.writable = true
	case ModeWriteCopy:
		protection = syscall.PAGE_WRITECOPY
		access = syscall.FILE_MAP_COPY
		mapping.writable = true
	default:
		return nil, &ErrorInvalidMode{Mode: mode}
	}
	if flags&FlagExecutable != 0 {
		protection <<= 4
		access |= syscall.FILE_MAP_EXECUTE
		mapping.executable = true
	}

	// Separate file handle needed to avoid errors on passed file external closing.
	var err error
	mapping.hProcess, err = syscall.GetCurrentProcess()
	if err != nil {
		return nil, os.NewSyscallError("GetCurrentProcess", err)
	}
	err = syscall.DuplicateHandle(
		mapping.hProcess, syscall.Handle(fd),
		mapping.hProcess, &mapping.hFile,
		0, true, syscall.DUPLICATE_SAME_ACCESS,
	)
	if err != nil {
		return nil, os.NewSyscallError("DuplicateHandle", err)
	}

	// Mapping area offset must be aligned by memory page size.
	pageSize := int64(os.Getpagesize())
	if pageSize < 0 {
		return nil, os.NewSyscallError("getpagesize", syscall.EINVAL)
	}
	outerOffset := offset / pageSize
	innerOffset := offset % pageSize
	mapping.alignedSize = uintptr(innerOffset) + size

	maxSize := uint64(outerOffset) + uint64(mapping.alignedSize)
	maxSizeHigh := uint32(maxSize >> 32)
	maxSizeLow := uint32(maxSize & uint64(math.MaxUint32))
	mapping.hMapping, err = syscall.CreateFileMapping(mapping.hFile, nil, protection, maxSizeHigh, maxSizeLow, nil)
	if err != nil {
		return nil, os.NewSyscallError("CreateFileMapping", err)
	}
	fileOffset := uint64(outerOffset)
	fileOffsetHigh := uint32(fileOffset >> 32)
	fileOffsetLow := uint32(fileOffset & uint64(math.MaxUint32))
	mapping.alignedAddress, err = syscall.MapViewOfFile(
		mapping.hMapping, access,
		fileOffsetHigh, fileOffsetLow, mapping.alignedSize,
	)
	if err != nil {
		return nil, os.NewSyscallError("MapViewOfFile", err)
	}
	mapping.address = mapping.alignedAddress + uintptr(innerOffset)
	mapping.size = size

	// Convert mapping to byte slice at required offset.
	var sliceHeader struct {
		data uintptr
		len  int
		cap  int
	}
	sliceHeader.data = mapping.address
	sliceHeader.len = int(mapping.size)
	sliceHeader.cap = sliceHeader.len
	mapping.data = *(*[]byte)(unsafe.Pointer(&sliceHeader))

	runtime.SetFinalizer(mapping, (*Mapping).Close)
	return mapping, nil
}

// Synchronize mapping with underlying file.
func (mapping *Mapping) Sync() error {
	if mapping.data == nil {
		return &ErrorClosed{}
	}
	if !mapping.writable {
		return &ErrorIllegalOperation{Operation: "sync"}
	}
	if err := syscall.FlushViewOfFile(mapping.alignedAddress, mapping.alignedSize); err != nil {
		return os.NewSyscallError("FlushViewOfFile", err)
	}
	if err := syscall.FlushFileBuffers(mapping.hFile); err != nil {
		return os.NewSyscallError("FlushFileBuffers", err)
	}
	return nil
}

// Close mapping.
// Implementation of io.Closer.
func (mapping *Mapping) Close() error {
	if mapping.data == nil {
		return &ErrorClosed{}
	}
	if mapping.writable {
		if err := mapping.Sync(); err != nil {
			return err
		}
	}
	if err := syscall.UnmapViewOfFile(mapping.alignedAddress); err != nil {
		return os.NewSyscallError("UnmapViewOfFile", err)
	}
	if err := syscall.CloseHandle(mapping.hMapping); err != nil {
		return os.NewSyscallError("CloseHandle", err)
	}
	if err := syscall.CloseHandle(mapping.hFile); err != nil {
		return os.NewSyscallError("CloseHandle", err)
	}
	*mapping = Mapping{}
	runtime.SetFinalizer(mapping, nil)
	return nil
}
