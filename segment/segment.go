// Package segment provides a data segment.
package segment

import (
	"encoding/binary"
	"io"
)

// ReadWriterAt is the interface that groups the basic io.ReadAt and io.WriteAt methods.
type ReadWriterAt interface {
	io.ReaderAt
	io.WriterAt
}

// Segment is a data segment.
// Supported data types are uint8, uint16, uint32 and uint64.
// All numeric values in the buffer are encoded using big-endian byte order.
type Segment struct {
	buf ReadWriterAt
}

// New returns a new data segment.
func New(buf ReadWriterAt) *Segment {
	return &Segment{
		buf: buf,
	}
}

func (seg *Segment) read(buf []byte, offset int64, index int) error {
	if n, err := seg.buf.ReadAt(buf, offset); err != nil {
		return err
	} else if n < len(buf) {
		return &ErrorPartialRead{Index: index, Offset: offset, NumBytes: n}
	}
	return nil
}

func (seg *Segment) write(buf []byte, offset int64, index int) error {
	if n, err := seg.buf.WriteAt(buf, offset); err != nil {
		return err
	} else if n < 1 {
		return &ErrorPartialWrite{Index: index, Offset: offset, NumBytes: n}
	}
	return nil
}

func (seg *Segment) next(buf []byte, offset *int64) {
	*offset += int64(len(buf))
}

// Get sequentially reads data from buffer starting from given offset into values pointed by v.
func (seg *Segment) Get(offset int64, v ...interface{}) error {
	for i, val := range v {
		switch val.(type) {
		default:
			return &ErrorUnsupportedType{Index: i}
		case *uint8:
			buf := make([]byte, 1)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			*val.(*uint8) = buf[0]
			seg.next(buf, &offset)
		case *uint16:
			buf := make([]byte, 2)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			*val.(*uint16) = binary.BigEndian.Uint16(buf)
			seg.next(buf, &offset)
		case *uint32:
			buf := make([]byte, 4)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			*val.(*uint32) = binary.BigEndian.Uint32(buf)
			seg.next(buf, &offset)
		case *uint64:
			buf := make([]byte, 8)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			*val.(*uint64) = binary.BigEndian.Uint64(buf)
			seg.next(buf, &offset)
		}
	}
	return nil
}

// Set sequentially writes values specified by v to the buffer starting from given offset.
func (seg *Segment) Set(offset int64, v ...interface{}) error {
	for i, val := range v {
		switch val.(type) {
		default:
			return &ErrorUnsupportedType{Index: i}
		case uint8:
			buf := make([]byte, 1)
			buf[0] = val.(uint8)
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint16:
			buf := make([]byte, 2)
			binary.BigEndian.PutUint16(buf, val.(uint16))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint32:
			buf := make([]byte, 4)
			binary.BigEndian.PutUint32(buf, val.(uint32))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint64:
			buf := make([]byte, 8)
			binary.BigEndian.PutUint64(buf, val.(uint64))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		}
	}
	return nil
}

// Inc sequentially increments values in the buffer starting from given offset using deltas specified by v.
func (seg *Segment) Inc(offset int64, v ...interface{}) error {
	for i, val := range v {
		switch val.(type) {
		default:
			return &ErrorUnsupportedType{Index: i}
		case uint8:
			buf := make([]byte, 1)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			buf[0] += val.(uint8)
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint16:
			buf := make([]byte, 2)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			binary.BigEndian.PutUint16(buf, binary.BigEndian.Uint16(buf)+val.(uint16))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint32:
			buf := make([]byte, 4)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			binary.BigEndian.PutUint32(buf, binary.BigEndian.Uint32(buf)+val.(uint32))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint64:
			buf := make([]byte, 8)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			binary.BigEndian.PutUint64(buf, binary.BigEndian.Uint64(buf)+val.(uint64))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		}
	}
	return nil
}

// Dec sequentially decrements values in the buffer starting from given offset using deltas specified by v.
func (seg *Segment) Dec(offset int64, v ...interface{}) error {
	for i, val := range v {
		switch val.(type) {
		default:
			return &ErrorUnsupportedType{Index: i}
		case uint8:
			buf := make([]byte, 1)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			buf[0] -= val.(uint8)
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint16:
			buf := make([]byte, 2)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			binary.BigEndian.PutUint16(buf, binary.BigEndian.Uint16(buf)-val.(uint16))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint32:
			buf := make([]byte, 4)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			binary.BigEndian.PutUint32(buf, binary.BigEndian.Uint32(buf)-val.(uint32))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		case uint64:
			buf := make([]byte, 8)
			if err := seg.read(buf, offset, i); err != nil {
				return err
			}
			binary.BigEndian.PutUint64(buf, binary.BigEndian.Uint64(buf)-val.(uint64))
			if err := seg.write(buf, offset, i); err != nil {
				return err
			}
			seg.next(buf, &offset)
		}
	}
	return nil
}
