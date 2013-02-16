package iobit

import (
	"encoding/binary"
	"errors"
	"io"
)

type Writer struct {
	dst   io.Writer
	data  [8]uint8
	cache uint64
	fill  uint
	err   error
}

type bigEndian struct{}
type littleEndian struct{}

var (
	ErrOverflow  = errors.New("bit overflow")
	ErrUnderflow = errors.New("bit underflow")
	BigEndian    bigEndian
	LittleEndian littleEndian
)

func NewWriter(dst io.Writer) *Writer {
	return &Writer{dst: dst}
}

type RawWriter struct {
	dst []uint8
}

func (w *RawWriter) Write(p []uint8) (int, error) {
	n := copy(w.dst, p)
	w.dst = w.dst[n:]
	if n != len(p) {
		return n, ErrOverflow
	}
	return n, nil
}

func NewRawWriter(dst []uint8) *Writer {
	return NewWriter(&RawWriter{dst: dst})
}

func (w *Writer) flushCache(bits uint) {
	if w.fill+bits <= 64 {
		return
	}
	binary.BigEndian.PutUint32(w.data[:], uint32(w.cache>>32))
	w.cache <<= 32
	w.fill -= 32
	w.write(w.data[:4])
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
		BigEndian.PutUint32(w, bits-32, uint32(val>>32))
		bits = 32
		val &= 0xFFFFFFFF
	}
	BigEndian.PutUint32(w, bits, uint32(val))
}

func (littleEndian) PutUint64(w *Writer, bits uint, val uint64) {
	if bits > 32 {
		LittleEndian.PutUint32(w, bits-32, uint32(val&0xFFFFFFFF))
		bits = 32
		val >>= 32
	}
	LittleEndian.PutUint32(w, bits, uint32(val))
}

func (w *Writer) write(src []uint8) (int, error) {
	n := 0
	if w.err == nil {
		n, w.err = w.dst.Write(src)
	}
	return n, w.err
}

func (w *Writer) Flush() error {
	idx := 0
	for w.fill >= 8 {
		w.data[idx] = uint8(w.cache >> 56)
		w.cache <<= 8
		w.fill -= 8
		idx++
	}
	if w.err == nil && w.fill != 0 {
		w.err = ErrUnderflow
	}
	w.write(w.data[:idx])
	return w.err
}

func (w *Writer) Write(p []uint8) (int, error) {
	w.Flush()
	return w.write(p)
}
