// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package iobit

import (
	"encoding/binary"
)

type Reader struct {
	src   []byte
	cache uint64
	idx   uint
	max   uint
	size  uint
}

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

func (r *Reader) IsBit() bool {
	skip := min(r.idx>>3, r.max+7)
	val := r.src[skip]
	val <<= r.idx - skip<<3
	val >>= 7
	r.idx += 1
	return val != 0
}

func (bigEndian) Uint32(r *Reader, bits uint) uint32 {
	skip := min(r.idx>>5<<2, r.max)
	val := binary.BigEndian.Uint64(r.src[skip:])
	val <<= r.idx - skip<<3
	val >>= 64 - bits
	r.idx += bits
	return uint32(val)
}

func (bigEndian) Int32(r *Reader, bits uint) int32 {
	skip := min(r.idx>>5<<2, r.max)
	val := int64(binary.BigEndian.Uint64(r.src[skip:]))
	val <<= r.idx - skip<<3
	val >>= 64 - bits // use sign-extension
	r.idx += bits
	return int32(val)
}

func (bigEndian) Uint64(r *Reader, bits uint) uint64 {
	var val uint64
	if bits > 32 {
		val = uint64(BigEndian.Uint32(r, 32))
		bits -= 32
		val <<= bits
	}
	return val + uint64(BigEndian.Uint32(r, bits))
}

func extend(v uint64, bits uint) int64 {
	m := ^uint64(0) << (bits - 1)
	return int64((v ^ m) - m)
}

func (bigEndian) Int64(r *Reader, bits uint) int64 {
	return extend(BigEndian.Uint64(r, bits), bits)
}

func bswap32(val uint32) uint32 {
	return val>>24 | val>>8&0xFF00 | val<<8&0xFF0000 | val<<24
}

func (littleEndian) Uint32(r *Reader, bits uint) uint32 {
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

func (littleEndian) Int32(r *Reader, bits uint) int32 {
	v := LittleEndian.Uint32(r, bits)
	return int32(extend(uint64(v), bits))
}

func (littleEndian) Uint64(r *Reader, bits uint) uint64 {
	var val uint64
	var shift uint
	if bits > 32 {
		val = uint64(LittleEndian.Uint32(r, 32))
		bits -= 32
		shift = 32
	}
	return val + uint64(LittleEndian.Uint32(r, bits))<<shift
}

func (littleEndian) Int64(r *Reader, bits uint) int64 {
	v := LittleEndian.Uint64(r, bits)
	return extend(v, bits)
}

func (r *Reader) Peek() *Reader {
	p := *r
	return &p
}

func (r *Reader) Skip(bits uint) {
	r.idx += bits
}

func (r *Reader) Index() uint {
	return r.idx
}

func (r *Reader) Bits() uint {
	return r.size<<3 - min(r.idx, r.size<<3)
}

func (r *Reader) Bytes() []byte {
	skip := min(r.idx>>3, r.size)
	last := r.size - skip
	if last == 0 {
		return r.src[0:0]
	}
	return r.src[skip:last]
}

func (r *Reader) Check() error {
	if r.idx > r.size<<3 {
		return ErrOverflow
	}
	return nil
}

func (r *Reader) Reset() {
	r.idx = 0
}
