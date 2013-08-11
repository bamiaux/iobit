// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package iobit

import (
	"errors"
)

// A writer wraps a raw byte array and provides multiple methoods to write data bit-by-bit
// Its methods don't return the usual error as it is too expensive.
// Instead, write errors can be checked with the Flush() method.
type Writer struct {
	dst   []byte
	cache uint64
	fill  uint
	idx   int
}

var (
	ErrOverflow  = errors.New("bit overflow")
	ErrUnderflow = errors.New("bit underflow")
)

// NewWriter returns a new writer writing to <dst> byte array.
func NewWriter(dst []byte) *Writer {
	return &Writer{dst: dst}
}

// PutUint32Be writes up to 32 <bits> from <val> in big-endian mode.
func (w *Writer) PutUint32Be(bits uint, val uint32) {
	u := uint64(val)
	// manually inlined until compiler improves
	if w.fill+bits > 64 {
		if w.idx+4 <= len(w.dst) {
			w.dst[w.idx+0] = byte(w.cache >> 56)
			w.dst[w.idx+1] = byte(w.cache >> 48)
			w.dst[w.idx+2] = byte(w.cache >> 40)
			w.dst[w.idx+3] = byte(w.cache >> 32)
		}
		w.idx += 4
		w.cache <<= 32
		w.fill -= 32
	}
	u &= ^(^uint64(0) << bits)
	u <<= 64 - w.fill - bits
	w.fill += bits
	w.cache |= u
}

// PutUint32Le writes up to 32 <bits> from <val> in little-endian mode.
func (w *Writer) PutUint32Le(bits uint, val uint32) {
	val = bswap32(val)
	left, right := bits&7, bits&0xF8
	sub := val >> (24 - right)
	// manually inlined until compiler improves
	if w.fill+bits > 64 {
		if w.idx+4 <= len(w.dst) {
			w.dst[w.idx+0] = byte(w.cache >> 56)
			w.dst[w.idx+1] = byte(w.cache >> 48)
			w.dst[w.idx+2] = byte(w.cache >> 40)
			w.dst[w.idx+3] = byte(w.cache >> 32)
		}
		w.idx += 4
		w.cache <<= 32
		w.fill -= 32
	}
	mask := ^uint32(0) << left
	sub &= ^mask
	val >>= 32 - bits
	val &= mask
	u := uint64(val + sub)
	u <<= 64 - w.fill - bits
	w.fill += bits
	w.cache |= u
}

// PutUint64Be writes up to 64 <bits> from <val> in big-endian mode.
func (w *Writer) PutUint64Be(bits uint, val uint64) {
	if bits > 32 {
		w.PutUint32Be(bits-32, uint32(val>>32))
		bits = 32
		val &= 0xFFFFFFFF
	}
	w.PutUint32Be(bits, uint32(val))
}

// PutUint64Le writes up to 64 <bits> from <val> in little-endian mode.
func (w *Writer) PutUint64Le(bits uint, val uint64) {
	if bits > 32 {
		w.PutUint32Le(32, uint32(val&0xFFFFFFFF))
		bits -= 32
		val >>= 32
	}
	w.PutUint32Le(bits, uint32(val))
}

// Flush flushes the writer to its array backend.
// Returns ErrUnderflow if the output is not byte-aligned.
// Returns ErrOverflow if the output array is too small.
func (w *Writer) Flush() error {
	for w.fill >= 8 && w.idx < len(w.dst) {
		w.dst[w.idx] = byte(w.cache >> 56)
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

// Write writes a whole byte slice at once from <p>.
// Returns an error if the writer is not byte-aligned.
func (w *Writer) Write(p []byte) (int, error) {
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

// Index returns the current writer position in bits.
func (w *Writer) Index() int {
	return w.idx<<3 + int(w.fill)
}

func imin(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// Bits returns the number of bits available to write.
func (w *Writer) Bits() int {
	size := len(w.dst)
	return size<<3 - imin(w.idx<<3+int(w.fill), size<<3)
}

// Bytes returns a byte array of what's left to write.
// Note that this array is 8-bit aligned even if the writer is not.
func (w *Writer) Bytes() []byte {
	skip := w.idx + int(w.fill>>3)
	if skip >= len(w.dst) {
		return w.dst[:0]
	}
	return w.dst[skip:len(w.dst)]
}

// Reset resets the writer to its initial position.
func (w *Writer) Reset() {
	w.fill = 0
	w.idx = 0
}
