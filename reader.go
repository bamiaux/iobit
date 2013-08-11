// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package iobit provides primitives for reading & writing bits

The main purpose of this library is to remove the need to write
custom bit-masks when reading or writing bitstreams, and to ease
maintenance. This is true especially when you need to read/write
data which is not aligned on bytes.

For example, with iobit you can read an MPEG-TS PCR like this:

    r := iobit.NewReader(buffer)
    base := r.Uint64Be(33)     // PCR base is 33-bits
    r.Skip(6)                  // 6-bits are reserved
    extension := r.Uint64Be(9) // PCR extension is 9-bits

instead of:

    base = uint64(buffer[0]) << 25
    base |= uint64(buffer[1]) << 17
    base |= uint64(buffer[2]) << 9
    base |= uint64(buffer[3]) << 1
    base |= buffer[4] >> 7
    extension := uint16(buffer[4] & 0x1) << 8
    extension |= buffer[5]

and write it like this:

    w := iobit.NewWriter(buffer)
    w.PutUint64Be(33, base)
    w.PutUint32Be(6, 0)
    w.PutUint32Be(9, extension)
*/
package iobit

import (
	"encoding/binary"
)

// A reader wraps a raw byte array and provides multiple methods to read and
// skip data bit-by-bit.
// Its methods don't return the usual error as it is too expensive.
// Instead, read errors can be checked with the Check() method
type Reader struct {
	src   []byte
	cache uint64
	idx   uint
	max   uint
	size  uint
}

// NewReader returns a new reader reading from <src> byte array.
func NewReader(src []byte) *Reader {
	if len(src) >= 8 {
		return &Reader{
			src:  src,
			max:  uint(len(src) - 8),
			size: uint(len(src)),
		}
	}
	clone := make([]byte, 8)
	copy(clone, src)
	return &Reader{
		src:  clone,
		size: uint(len(src)),
	}
}

func min(a, b uint) uint {
	if a > b {
		return b
	}
	return a
}

// IsBit reads the next bit and returns whether it is set.
func (r *Reader) IsBit() bool {
	skip := min(r.idx>>3, r.max+7)
	val := r.src[skip]
	val <<= r.idx - skip<<3
	val >>= 7
	r.idx += 1
	return val != 0
}

// Uint32Be reads up to 32 <bits> from <r> in big-endian mode.
// It returns a 32-bit unsigned integer.
func (r *Reader) Uint32Be(bits uint) uint32 {
	skip := min(r.idx>>5<<2, r.max)
	val := binary.BigEndian.Uint64(r.src[skip:])
	val <<= r.idx - skip<<3
	val >>= 64 - bits
	r.idx += bits
	return uint32(val)
}

// Int32Be reads up to 32 <bits> from <r> in big-endian mode.
// It returns a 32-bit signed integer.
func (r *Reader) Int32Be(bits uint) int32 {
	skip := min(r.idx>>5<<2, r.max)
	val := int64(binary.BigEndian.Uint64(r.src[skip:]))
	val <<= r.idx - skip<<3
	val >>= 64 - bits // use sign-extension
	r.idx += bits
	return int32(val)
}

// Uint64Be reads up to 64 <bits> from <r> in big-endian mode.
// It returns a 64-bit unsigned integer.
func (r *Reader) Uint64Be(bits uint) uint64 {
	var val uint64
	if bits > 32 {
		val = uint64(r.Uint32Be(32))
		bits -= 32
		val <<= bits
	}
	return val + uint64(r.Uint32Be(bits))
}

func extend(v uint64, bits uint) int64 {
	m := ^uint64(0) << (bits - 1)
	return int64((v ^ m) - m)
}

// Int64Be reads up to 64 <bits> from <r> in big-endian mode.
// It returns a 64-bit signed integer.
func (r *Reader) Int64Be(bits uint) int64 {
	return extend(r.Uint64Be(bits), bits)
}

func bswap32(val uint32) uint32 {
	return val>>24 | val>>8&0xFF00 | val<<8&0xFF0000 | val<<24
}

// Uint32Le reads up to 32 <bits> from <r> in little-endian mode.
// It returns a 32-bit unsigned integer.
func (r *Reader) Uint32Le(bits uint) uint32 {
	skip := min(r.idx>>5<<2, r.max)
	offset := r.idx - skip<<3
	r.idx += bits
	big := binary.BigEndian.Uint64(r.src[skip:])
	val := bswap32(uint32(big << offset >> 32))
	left, right := bits&7, bits&0xF8
	sub := val >> (8 - left)
	sub &= ^(^uint32(0) << left) << right
	val &= ^(^uint32(0) << right)
	return sub + val
}

// Int32Le reads up to 32 <bits> from <r> in little-endian mode.
// It returns a 32-bit signed integer.
func (r *Reader) Int32Le(bits uint) int32 {
	return int32(extend(uint64(r.Uint32Le(bits)), bits))
}

// Uint64Le reads up to 64 <bits> from <r> in little-endian mode.
// It returns a 64-bit unsigned integer.
func (r *Reader) Uint64Le(bits uint) uint64 {
	var val uint64
	var shift uint
	if bits > 32 {
		val = uint64(r.Uint32Le(32))
		bits -= 32
		shift = 32
	}
	return val + uint64(r.Uint32Le(bits))<<shift
}

// Int64Le reads up to 64 <bits> from <r> in little-endian mode.
// It returns a 64-bit signed integer.
func (r *Reader) Int64Le(bits uint) int64 {
	return extend(r.Uint64Le(bits), bits)
}

// Peek returns a new reader at the same position as the original reader.
// Any read done on this read will not move the original reader.
func (r *Reader) Peek() *Reader {
	p := *r
	return &p
}

// Skip skips n <bits> from the reader.
func (r *Reader) Skip(bits uint) {
	r.idx += bits
}

// Index returns the current reader position in bits.
func (r *Reader) Index() uint {
	return r.idx
}

// Bits returns the number of bits left to read.
func (r *Reader) Bits() uint {
	return r.size<<3 - min(r.idx, r.size<<3)
}

// Bytes returns a byte array of what's left to read.
// Note that this array is 8-bit aligned even if the reader is not.
func (r *Reader) Bytes() []byte {
	skip := r.idx >> 3
	if skip >= r.size {
		return r.src[:0]
	}
	return r.src[skip:r.size]
}

// Check returns whether the reader encountered an error on previous methods.
func (r *Reader) Check() error {
	if r.idx > r.size<<3 {
		return ErrOverflow
	}
	return nil
}

// Reset resets the reader to its initial position.
func (r *Reader) Reset() {
	r.idx = 0
}
