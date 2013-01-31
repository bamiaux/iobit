package iobit

import (
	"encoding/binary"
	"errors"
	"io"
)

type Writer struct {
	cache uint64
	data  []uint8
	dst   io.Writer
	index int
	fill  uint
	err   error
}

func NewWriterSize(dst io.Writer, size int) *Writer {
	return &Writer{
		data: make([]uint8, size),
		dst:  dst,
	}
}

func NewWriter(dst io.Writer) *Writer {
	return NewWriterSize(dst, 64)
}

func (w *Writer) WriteBits(bits uint, val uint32) {
	if w.fill+bits > 64 {
		binary.BigEndian.PutUint32(w.data[w.index:], uint32(w.cache>>32))
		w.cache <<= 32
		w.fill -= 32
		w.index += 4
		if w.index+4 > len(w.data) {
			w.write()
		}
	}
	u := uint64(val)
	u &= ^(^uint64(0) << bits)
	u <<= 64 - w.fill - bits
	w.cache |= u
	w.fill += bits
}

func (w *Writer) Write64Bits(bits uint, val uint64) {
	if bits > 32 {
		w.WriteBits(bits-32, uint32(val>>32))
		bits = 32
		val &= 0xFFFFFFFF
	}
	w.WriteBits(bits, uint32(val))
}

func (w *Writer) write() {
	if w.err == nil {
		_, w.err = w.dst.Write(w.data[:w.index])
	}
	w.index = 0
}

func (w *Writer) Flush() error {
	for w.fill >= 8 {
		w.data[w.index] = uint8(w.cache >> 56)
		w.cache <<= 8
		w.fill -= 8
		w.index++
	}
	if w.fill != 0 {
		w.err = errors.New("iobit: unable to flush unaligned output")
	}
	w.write()
	return w.err
}

func (w *Writer) Write(p []uint8) (int, error) {
	err := w.Flush()
	if err != nil {
		return 0, err
	}
	return w.dst.Write(p)
}
