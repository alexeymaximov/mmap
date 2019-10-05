package segment

import (
	"os"

	"github.com/alexeymaximov/mmap"
)

// MappedSegment is a data segment on top of the memory mapping.
type MappedSegment struct {
	*mmap.Mapping
	*Segment
}

// NewMapped returns a new data segment on top of the memory mapping.
func NewMapped(m *mmap.Mapping) *MappedSegment {
	return &MappedSegment{
		Mapping: m,
		Segment: New(m),
	}
}

// NewFile prepares a data segment file, calls init function if file was just created
// and returns a new data segment on top of the mapping of file into the memory.
func NewFile(name string, perm os.FileMode, size uintptr, init func(seg *MappedSegment) error) (*MappedSegment, error) {
	m, created, err := func() (*mmap.Mapping, bool, error) {
		created := false
		if _, err := os.Stat(name); err != nil && os.IsNotExist(err) {
			created = true
		}
		f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, perm)
		if err != nil {
			return nil, false, err
		}
		defer f.Close()
		if err := f.Truncate(int64(size)); err != nil {
			return nil, false, err
		}
		m, err := mmap.New(f.Fd(), 0, size, mmap.ModeReadWrite, 0)
		if err != nil {
			return nil, false, err
		}
		return m, created, nil
	}()
	if err != nil {
		return nil, err
	}
	seg := NewMapped(m)
	if created && init != nil {
		if err := init(seg); err != nil {
			m.Close()
			os.Remove(name)
			return nil, err
		}
	}
	return seg, nil
}

// MappedSegmentTransaction is a data segment on top of the memory mapping transaction.
type MappedSegmentTransaction struct {
	*mmap.Transaction
	*Segment
}

// Begin starts a transaction.
func (seg *MappedSegment) Begin(offset int64, length uintptr) (*MappedSegmentTransaction, error) {
	tx, err := seg.Mapping.Begin(offset, length)
	if err != nil {
		return nil, err
	}
	return &MappedSegmentTransaction{
		Transaction: tx,
		Segment:     New(tx),
	}, nil
}
