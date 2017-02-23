// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package iobit

import (
	"testing"
)

func testReads(t *testing.T, op ReadTestOp) {
	src := makeSource(1 << 16)
	max := len(src) * 8
	for i := 32; i > 0; i >>= 1 {
		dst := make([]byte, len(src))
		r := NewReader(src)
		w := NewWriter(dst)
		for read := 0; read < max; {
			bits := getNumBits(read, max, 64, i)
			op(w, r, uint(bits))
			read += bits
		}
		flushCheck(t, w)
		compare(t, src, dst)
	}
}

func bitLoop(w *Writer, r *Reader, bits uint) {
	for i := uint(0); i < bits; i++ {
		v := uint32(0)
		if r.IsBit() {
			v = 1
		}
		w.PutUint32(1, v)
	}
}

func bigUint64Loop(w *Writer, r *Reader, bits uint) {
	w.PutUint64(bits, r.Uint64(bits))
}

func bigInt64Loop(w *Writer, r *Reader, bits uint) {
	w.PutUint64(bits, uint64(r.Int64(bits)))
}

type ReadTestOp func(w *Writer, r *Reader, bits uint)

func TestBitReads(t *testing.T)       { testReads(t, bitLoop) }
func TestBigUint64Reads(t *testing.T) { testReads(t, bigUint64Loop) }
func TestBigInt64Reads(t *testing.T)  { testReads(t, bigInt64Loop) }

func TestSigned(t *testing.T) {
	big := []byte{0x7E}
	r := NewReader(big)
	expect(t, int32(0), r.Int32(1))
	expect(t, int32(-1), r.Int32(1))
	expect(t, int32(-1), r.Int32(5))
	expect(t, int32(0), r.Int32(1))
	big = []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xE0}
	r = NewReader(big)
	expect(t, int64(0), r.Int64(1))
	expect(t, int64(-1), r.Int64(1))
	expect(t, int64(-1), r.Int64(33))
	expect(t, int64(0), r.Int64(5))
	lil := []byte{0x7F, 0xFE}
	r = NewReader(lil)
	expect(t, int32(0), r.Int32Le(1))
	expect(t, int32(-1), r.Int32Le(1))
	expect(t, int32(-1), r.Int32Le(13))
	expect(t, int32(0), r.Int32Le(1))
	lil = []byte{0x7F, 0x7F, 0xFF, 0xFF, 0xE0}
	r = NewReader(lil)
	expect(t, int64(0), r.Int64Le(1))
	expect(t, int64(-1), r.Int64Le(1))
	expect(t, int64(-3), r.Int64Le(33))
	expect(t, int64(0), r.Int64Le(5))
}

func TestReadHelpers(t *testing.T) {
	buf := []byte{0x41}
	r := NewReader(buf[:])
	expect(t, uint(8), r.Bits())
	r.Skip(1)
	expect(t, uint(1), r.Index())
	expect(t, uint(7), r.Bits())
	for i := 0; i < 8; i++ {
		p := r.Peek()
		expect(t, true, p.IsBit())
		expect(t, false, p.IsBit())
	}
	expect(t, true, r.IsBit())
	for i := 0; i < 5; i++ {
		expect(t, false, r.IsBit())
	}
	expect(t, true, r.IsBit())
	expect(t, uint(8), r.Index())
	expect(t, uint(0), r.Bits())
	expect(t, 0, len(r.Bytes()))
	expect(t, nil, r.Check())
	r.Skip(1)
	expect(t, uint(9), r.Index())
	expect(t, uint(0), r.Bits())
	expect(t, 0, len(r.Bytes()))
	expect(t, ErrOverflow, r.Check())
}

func TestBadSliceRead(t *testing.T) {
	buf := []byte{0x01, 0x02, 0x03}
	r := NewReader(buf[:])
	r.Skip(8)
	compare(t, r.Bytes(), buf[1:])
	r.Skip(16)
	expect(t, 0, len(r.Bytes()))
	r.Skip(1)
	expect(t, 0, len(r.Bytes()))
}

var Output int64

type ReadBench struct {
	name string
	op   func(r *Reader) int64
}

func BenchmarkReads(b *testing.B) {
	buf := makeSource(32)
	r := NewReader(buf)
	b.ResetTimer()
	bitbench := ReadBench{"bit", func(r *Reader) int64 {
		if r.IsBit() {
			return 1
		}
		return 0
	}}
	for _, v := range []ReadBench{
		bitbench,
		{"byte", func(r *Reader) int64 { return int64(r.Uint32(8)) }},
		{"le16", func(r *Reader) int64 { return int64(r.Uint32Le(16)) }},
		{"be16", func(r *Reader) int64 { return int64(r.Uint32(16)) }},
		{"le32", func(r *Reader) int64 { return int64(r.Uint32Le(32)) }},
		{"be32", func(r *Reader) int64 { return int64(r.Uint32(32)) }},
		{"le64", func(r *Reader) int64 { return int64(r.Uint64Le(64)) }},
		{"be64", func(r *Reader) int64 { return int64(r.Uint64(64)) }},
		{"u8 7bits", func(r *Reader) int64 { return int64(r.Uint32(7)) }},
		{"i8 7bits", func(r *Reader) int64 { return int64(r.Int32(7)) }},
		{"u16 15bits", func(r *Reader) int64 { return int64(r.Uint32(15)) }},
		{"i16 15bits", func(r *Reader) int64 { return int64(r.Int32(15)) }},
		{"u32 31bits", func(r *Reader) int64 { return int64(r.Uint32(31)) }},
		{"i32 31bits", func(r *Reader) int64 { return int64(r.Int32(31)) }},
		{"u64 63bits", func(r *Reader) int64 { return int64(r.Uint64(63)) }},
		{"i64 63bits", func(r *Reader) int64 { return int64(r.Int64(63)) }},
	} {
		b.Run(v.name, func(bb *testing.B) {
			bb.SetBytes(int64(len(buf)))
			for i := 0; i < bb.N; i++ {
				r.Reset()
				for r.Bits() > 0 {
					Output += v.op(r)
				}
			}
		})
	}
}
