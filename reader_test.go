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

func bigUint64Loop(w *Writer, r *Reader, bits uint) {
	w.PutUint64Be(bits, r.Uint64Be(bits))
}

func bigInt64Loop(w *Writer, r *Reader, bits uint) {
	w.PutUint64Be(bits, uint64(r.Int64Be(bits)))
}

func littleUint64Loop(w *Writer, r *Reader, bits uint) {
	w.PutUint64Le(bits, r.Uint64Le(bits))
}

func littleInt64Loop(w *Writer, r *Reader, bits uint) {
	w.PutUint64Le(bits, uint64(r.Int64Le(bits)))
}

type ReadTestOp func(w *Writer, r *Reader, bits uint)

func TestBigUint64Reads(t *testing.T)    { testReads(t, bigUint64Loop) }
func TestBigInt64Reads(t *testing.T)     { testReads(t, bigInt64Loop) }
func TestLittleUint64Reads(t *testing.T) { testReads(t, littleUint64Loop) }
func TestLittleInt64Reads(t *testing.T)  { testReads(t, littleInt64Loop) }

func TestSigned(t *testing.T) {
	big := []byte{0x7E}
	r := NewReader(big)
	expect(t, int32(0), r.Int32Be(1))
	expect(t, int32(-1), r.Int32Be(1))
	expect(t, int32(-1), r.Int32Be(5))
	expect(t, int32(0), r.Int32Be(1))
	big = []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xE0}
	r = NewReader(big)
	expect(t, int64(0), r.Int64Be(1))
	expect(t, int64(-1), r.Int64Be(1))
	expect(t, int64(-1), r.Int64Be(33))
	expect(t, int64(0), r.Int64Be(5))
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

func bigReadUint32(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Uint32Be(bits[j])
	}
}

func bigReadUint64(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Uint64Be(bits[j])
	}
}

func bigReadInt32(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Int32Be(bits[j])
	}
}

func bigReadInt64(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Int64Be(bits[j])
	}
}

func littleReadUint32(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Uint32Le(bits[j])
	}
}

func littleReadUint64(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Uint64Le(bits[j])
	}
}

func littleReadInt32(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Int32Le(bits[j])
	}
}

func littleReadInt64(r *Reader, bits []uint, last int) {
	for j := 0; j < last; j++ {
		r.Int64Le(bits[j])
	}
}

type ReadOp func(*Reader, []uint, int)

func benchmarkReads(b *testing.B, op ReadOp, chunk, align int) {
	size := 1 << 12
	buf, bits, _, last := prepareBenchmark(size, chunk, align)
	b.SetBytes(int64(len(buf)))
	r := NewReader(buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reset()
		op(r, bits, last)
	}
}

func BenchmarkBigEndianReadUint32Align1(b *testing.B)     { benchmarkReads(b, bigReadUint32, 32, 1) }
func BenchmarkBigEndianReadUint32Align32(b *testing.B)    { benchmarkReads(b, bigReadUint32, 32, 32) }
func BenchmarkBigEndianReadUint64Align1(b *testing.B)     { benchmarkReads(b, bigReadUint64, 64, 1) }
func BenchmarkBigEndianReadUint64Align32(b *testing.B)    { benchmarkReads(b, bigReadUint64, 64, 32) }
func BenchmarkBigEndianReadUint64Align64(b *testing.B)    { benchmarkReads(b, bigReadUint64, 64, 64) }
func BenchmarkBigEndianReadInt32Align1(b *testing.B)      { benchmarkReads(b, bigReadInt32, 32, 1) }
func BenchmarkBigEndianReadInt32Align32(b *testing.B)     { benchmarkReads(b, bigReadInt32, 32, 32) }
func BenchmarkBigEndianReadInt64Align1(b *testing.B)      { benchmarkReads(b, bigReadInt64, 64, 1) }
func BenchmarkBigEndianReadInt64Align32(b *testing.B)     { benchmarkReads(b, bigReadInt64, 64, 32) }
func BenchmarkBigEndianReadInt64Align64(b *testing.B)     { benchmarkReads(b, bigReadInt64, 64, 64) }
func BenchmarkLittleEndianReadUint32Align1(b *testing.B)  { benchmarkReads(b, littleReadUint32, 32, 1) }
func BenchmarkLittleEndianReadUint32Align32(b *testing.B) { benchmarkReads(b, littleReadUint32, 32, 32) }
func BenchmarkLittleEndianReadUint64Align1(b *testing.B)  { benchmarkReads(b, littleReadUint64, 64, 1) }
func BenchmarkLittleEndianReadUint64Align32(b *testing.B) { benchmarkReads(b, littleReadUint64, 64, 32) }
func BenchmarkLittleEndianReadUint64Align64(b *testing.B) { benchmarkReads(b, littleReadUint64, 64, 64) }
func BenchmarkLittleEndianReadInt32Align1(b *testing.B)   { benchmarkReads(b, littleReadInt32, 32, 1) }
func BenchmarkLittleEndianReadInt32Align32(b *testing.B)  { benchmarkReads(b, littleReadInt32, 32, 32) }
func BenchmarkLittleEndianReadInt64Align1(b *testing.B)   { benchmarkReads(b, littleReadInt64, 64, 1) }
func BenchmarkLittleEndianReadInt64Align32(b *testing.B)  { benchmarkReads(b, littleReadInt64, 64, 32) }
func BenchmarkLittleEndianReadInt64Align64(b *testing.B)  { benchmarkReads(b, littleReadInt64, 64, 64) }
