package iobit

import (
	"encoding/binary"
	"errors"
	"io"
)

type Writer struct {
	cache uint64
	dst   io.Writer
	fill  uint
	err   error
}

type bigEndian struct{}
type littleEndian struct{}

var (
	ErrUnderflow = errors.New("bit underflow")
	BigEndian    bigEndian
	LittleEndian littleEndian
)

func NewWriter(dst io.Writer) *Writer {
	return &Writer{dst: dst}
}

func (w *Writer) flushCache(bits uint) {
	if w.fill+bits <= 64 {
		return
	}
	var data [4]uint8
	binary.BigEndian.PutUint32(data[:], uint32(w.cache>>32))
	w.cache <<= 32
	w.fill -= 32
	w.write(data[:])
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

func (w *Writer) write(data []uint8) {
	if w.err == nil {
		_, w.err = w.dst.Write(data)
	}
}

func (w *Writer) Flush() error {
	var data [8]uint8
	idx := 0
	for w.fill >= 8 {
		data[idx] = uint8(w.cache >> 56)
		w.cache <<= 8
		w.fill -= 8
		idx++
	}
	if w.fill != 0 {
		w.err = ErrUnderflow
	}
	w.write(data[:])
	return w.err
}

func (w *Writer) Write(p []uint8) (int, error) {
	err := w.Flush()
	if err != nil {
		return 0, err
	}
	return w.dst.Write(p)
}
