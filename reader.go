// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package iobit provides primitives for reading & writing bits

The main purpose of this library is to remove the need to write
custom bit-masks when reading or writing bitstreams, and to ease
maintenance. This is true especially when you need to read/write
data which is not aligned on bytes.

Errors are sticky so you can check for errors after a chunk of
meaningful work rather than after every operation.

For example, with iobit you can read an MPEG-TS PCR like this:

    r := iobit.NewReader(buffer)
    base := r.Uint64(33)     // PCR base is 33-bits
    r.Skip(6)                // 6-bits are reserved
    extension := r.Uint64(9) // PCR extension is 9-bits

instead of:

    base  = uint64(buffer[0]) << 25
    base |= uint64(buffer[1]) << 17
    base |= uint64(buffer[2]) << 9
    base |= uint64(buffer[3]) << 1
    base |= buffer[4] >> 7
    extension := uint16(buffer[4] & 0x1) << 8
    extension |= buffer[5]

and write it like this:

    w := iobit.NewWriter(buffer)
    w.PutUint64(33, base)
    w.PutUint32(6, 0)
    w.PutUint32(9, extension)
*/
package iobit

import (
	"encoding/binary"
)

// Reader wraps a raw byte array and provides multiple methods to read and
// skip data bit-by-bit.
// Its methods don't return the usual error as it is too expensive.
// Instead, read errors can be checked with the Check() method
type Reader struct {
	src  []byte
	idx  uint
	max  uint
	size uint
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

func bswap16(v uint16) uint16 {
	return v>>8 | v<<8
}

func bswap32(val uint32) uint32 {
	return uint32(bswap16(uint16(val>>16))) | uint32(bswap16(uint16(val&0xFFFF)))<<16
}

// IsBit reads the next bit as a boolean.
func (r *Reader) Bit() bool {
	skip := min(r.idx>>3, r.max+7)
	val := r.src[skip]
	val <<= r.idx - skip<<3
	val >>= 7
	r.idx += 1
	return val != 0
}

func (r *Reader) get64(bits uint) uint64 {
	skip := min(r.idx>>5<<2, r.max)
	val := binary.BigEndian.Uint64(r.src[skip:])
	val <<= r.idx - skip<<3
	r.idx += bits
	return val
}

func (r *Reader) read32(bits uint) uint64 {
	return r.get64(bits) >> (64 - bits)
}

func (r *Reader) read32i(bits uint) int64 {
	// we need sign extension
	return int64(r.get64(bits)) >> (64 - bits)
}

// Byte reads one byte.
func (r *Reader) Byte() uint8 {
	return uint8(r.read32(8))
}

// Uint8 reads up to 8 unsigned bits in big-endian order.
func (r *Reader) Uint8(bits uint) uint8 {
	return uint8(r.read32(bits))
}

// Int8 reads up to 8 signed bits in big-endian order.
func (r *Reader) Int8(bits uint) int8 {
	return int8(r.read32i(bits))
}

// Be16 reads 16 unsigned bits in big-endian order.
func (r *Reader) Be16() uint16 {
	return uint16(r.read32(16))
}

// Uint16 reads up to 16 unsigned bits in big-endian order.
func (r *Reader) Uint16(bits uint) uint16 {
	return uint16(r.read32(bits))
}

// Int16 reads up to 16 signed bits in big-endian order.
func (r *Reader) Int16(bits uint) int16 {
	return int16(r.read32i(bits))
}

// Le16 reads 16 unsigned bits in litle-endian order.
func (r *Reader) Le16() uint16 {
	return bswap16(r.Be16())
}

// Be32 reads 32 unsigned bits in big-endian order.
func (r *Reader) Be32() uint32 {
	return uint32(r.read32(32))
}

// Uint32 reads up to 32 unsigned bits in big-endian order.
func (r *Reader) Uint32(bits uint) uint32 {
	return uint32(r.read32(bits))
}

// Int32 reads up to 32 signed bits in big-endian order.
func (r *Reader) Int32(bits uint) int32 {
	return int32(r.read32i(bits))
}

// Le32 reads 32 unsigned bits in little-endian order.
func (r *Reader) Le32() uint32 {
	return bswap32(r.Be32())
}

// Be64 reads 64 unsigned bits in big-endian order.
func (r *Reader) Be64() uint64 {
	v := r.Be32()
	return uint64(v)<<32 | uint64(r.Be32())
}

// Le64 reads 64 unsigned bits in little-endian order.
func (r *Reader) Le64() uint64 {
	low := r.Be32()
	high := r.Be32()
	return uint64(bswap32(high))<<32 | uint64(bswap32(low))
}

// Uint64 reads up to 64 unsigned bits in big-endian order.
func (r *Reader) Uint64(bits uint) uint64 {
	var val uint64
	if bits > 32 {
		val = r.read32(32)
		bits -= 32
		val <<= bits
	}
	return val | r.read32(bits)
}

// Int64 reads up to 64 signed bits in big-endian order.
func (r *Reader) Int64(bits uint) int64 {
	if bits <= 32 {
		return r.read32i(bits)
	}
	val := r.read32i(32)
	bits -= 32
	return val<<bits | int64(r.read32(bits))
}

// Peek returns a reader copy.
// Useful to read data without advancing the original reader.
func (r *Reader) Peek() *Reader {
	p := *r
	return &p
}

// Skip skips n bits.
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

// Bytes returns a slice of the contents of the unread reader portion.
// Note that this slice is byte aligned even if the reader is not.
func (r *Reader) Bytes() []byte {
	skip := r.idx >> 3
	if skip >= r.size {
		return r.src[:0]
	}
	return r.src[skip:r.size]
}

// Check returns whether the reader encountered an error.
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
