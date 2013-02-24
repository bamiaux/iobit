package iobit

import (
	"encoding/binary"
	"errors"
)

type Writer struct {
	dst   []uint8
	cache uint64
	fill  uint
	idx   int
}

type bigEndian struct{}
type littleEndian struct{}

var (
	ErrOverflow  = errors.New("bit overflow")
	ErrUnderflow = errors.New("bit underflow")
	BigEndian    bigEndian
	LittleEndian littleEndian
)

func NewWriter(dst []uint8) *Writer {
	return &Writer{dst: dst}
}

func (w *Writer) flushCache(bits uint) {
	if w.fill+bits <= 64 {
		return
	}
	if w.idx+4 <= len(w.dst) {
		binary.BigEndian.PutUint32(w.dst[w.idx:], uint32(w.cache>>32))
	}
	w.idx += 4
	w.cache <<= 32
	w.fill -= 32
}

func (w *Writer) writeCache(bits uint, val uint32) {
	u := uint64(val)
	u &= ^(^uint64(0) << bits)
	u <<= 64 - w.fill - bits
	w.cache |= u
	w.fill += bits
}

func (bigEndian) PutUint32(w *Writer, bits uint, val uint32) {
	w.flushCache(bits)
	w.writeCache(bits, val)
}

func (littleEndian) PutUint32(w *Writer, bits uint, val uint32) {
	w.flushCache(bits)
	for bits > 8 {
		w.writeCache(8, val)
		val >>= 8
		bits -= 8
	}
	w.writeCache(bits, val)
}

func (bigEndian) PutUint64(w *Writer, bits uint, val uint64) {
	if bits > 32 {
		bits -= 32
		BigEndian.PutUint32(w, 32, uint32(val>>bits))
		val &= 0xFFFFFFFF
	}
	BigEndian.PutUint32(w, bits, uint32(val))
}

func (littleEndian) PutUint64(w *Writer, bits uint, val uint64) {
	if bits > 32 {
		LittleEndian.PutUint32(w, 32, uint32(val&0xFFFFFFFF))
		bits -= 32
		val >>= 32
	}
	LittleEndian.PutUint32(w, bits, uint32(val))
}

func (w *Writer) Flush() error {
	for w.fill >= 8 && w.idx < len(w.dst) {
		w.dst[w.idx] = uint8(w.cache >> 56)
		w.idx += 1
		w.cache <<= 8
		w.fill -= 8
	}
	if w.idx+int(w.fill) > len(w.dst) {
		return ErrOverflow
	}
	if w.fill != 0 {
		return ErrUnderflow
	}
	return nil
}

func (w *Writer) Write(p []uint8) (int, error) {
	err := w.Flush()
	if err != nil {
		return 0, err
	}
	n := 0
	if w.idx < len(w.dst) {
		n = copy(w.dst[w.idx:], p)
	}
	w.idx += len(p)
	if n != len(p) {
		return n, ErrOverflow
	}
	return n, nil
}
