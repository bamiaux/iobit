// Copyright 2013 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package iobit

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
)

func getNumBits(read, max, chunk, align int) int {
	bits := 1
	if align != chunk {
		bits += rand.Intn(chunk / align)
	}
	bits *= align
	if read+bits > max {
		bits = max - read
	}
	if bits > chunk {
		panic("too many bits")
	}
	return bits
}

func makeSource(size int) []byte {
	src := make([]byte, size)
	for i := range src {
		src[i] = byte(rand.Intn(0xFF))
	}
	return src[:]
}

func flushCheck(t *testing.T, w *Writer) {
	err := w.Flush()
	if err != nil {
		t.Fatal("unexpected error during flush", err)
	}
}

func compare(t *testing.T, src, dst []byte) {
	if bytes.Equal(src, dst) {
		return
	}
	t.Log(hex.Dump(src))
	t.Log(hex.Dump(dst))
	t.Fatal("invalid output")
}

func testWrites(w *Writer, t *testing.T, align int, src []byte) {
	max := len(src) * 8
	for read := 0; read < max; {
		bits := getNumBits(read, max, 32, align)
		idx := read >> 3
		fill := read - idx*8
		if idx*8 > max-64 {
			rewind := max - 64
			fill += idx*8 - rewind
			idx = rewind >> 3
		}
		block := binary.BigEndian.Uint64(src[idx:])
		block >>= uint(64 - bits - fill)
		value := uint32(block & 0xFFFFFFFF)
		w.PutUint32Be(uint(bits), value)
		read += bits
	}
	flushCheck(t, w)
}

func TestWrites(t *testing.T) {
	src := makeSource(512)
	dst := make([]byte, len(src))
	for i := 32; i > 0; i >>= 1 {
		w := NewWriter(dst)
		testWrites(w, t, i, src)
		compare(t, src, dst)
	}
}

func TestSmall64BigEndianWrite(t *testing.T) {
	buf := make([]byte, 5)
	w := NewWriter(buf)
	w.PutUint64Be(33, 0xFFFFFFFE00000001)
	w.PutUint32Be(7, 0)
	w.Flush()
	compare(t, buf, []byte{0x00, 0x00, 0x00, 0x00, 0x80})
}

func TestSmall64LittleEndianWrite(t *testing.T) {
	buf := make([]byte, 5)
	w := NewWriter(buf)
	w.PutUint64Le(33, 0xFFFFFFFE00000001)
	w.PutUint32Le(7, 0)
	w.Flush()
	compare(t, buf, []byte{0x01, 0x00, 0x00, 0x00, 0x00})
}

func TestBigEndianWrite(t *testing.T) {
	buf := make([]byte, 8)
	w := NewWriter(buf)
	w.PutUint64Be(64, 0x0123456789ABCDEF)
	w.Flush()
	compare(t, buf, []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF})
}

func TestLittleEndianWrite(t *testing.T) {
	buf := make([]byte, 8)
	w := NewWriter(buf)
	w.PutUint64Le(64, 0x0123456789ABCDEF)
	w.Flush()
	compare(t, buf, []byte{0xEF, 0xCD, 0xAB, 0x89, 0x67, 0x45, 0x23, 0x01})
}

func checkError(t *testing.T, expected, actual error) {
	if actual != expected {
		t.Fatal("expecting", expected, "got", actual)
	}
}

func TestFlushErrors(t *testing.T) {
	buf := make([]byte, 2)

	w := NewWriter(buf)
	w.PutUint32Be(9, 0)
	checkError(t, ErrUnderflow, w.Flush())

	w = NewWriter(buf)
	w.PutUint32Be(16, 0)
	checkError(t, nil, w.Flush())

	w = NewWriter(buf)
	w.PutUint32Be(17, 0)
	checkError(t, ErrOverflow, w.Flush())
}

func expect(t *testing.T, a, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}
	typea := reflect.TypeOf(a)
	typeb := reflect.TypeOf(b)
	_, file, line, _ := runtime.Caller(1)
	t.Fatalf("%v:%v expectation failed %v(%v) != %v(%v)\n",
		file, line, typea, a, typeb, b)
}

func TestWriteHelpers(t *testing.T) {
	buf := []byte{0x00}
	w := NewWriter(buf[:])
	expect(t, int(8), w.Bits())
	w.PutUint32Be(1, 0)
	expect(t, int(1), w.Index())
	expect(t, int(7), w.Bits())
	w.PutUint32Be(1, 1)
	w.PutUint32Be(5, 0)
	w.PutUint32Be(1, 1)
	err := w.Flush()
	expect(t, buf, []byte{0x41})
	expect(t, int(8), w.Index())
	expect(t, int(0), w.Bits())
	expect(t, 0, len(w.Bytes()))
	expect(t, nil, err)
	w.PutUint32Be(1, 0)
	expect(t, int(9), w.Index())
	expect(t, int(0), w.Bits())
	expect(t, 0, len(w.Bytes()))
	expect(t, ErrOverflow, w.Flush())
}

func TestBadSlices(t *testing.T) {
	dst := []byte{0x00, 0x01, 0x02}
	w := NewWriter(dst[:])
	compare(t, w.Bytes(), dst[:])
	w.PutUint32Be(8, 0)
	compare(t, w.Bytes(), dst[1:])
	w.PutUint32Be(16, 0)
	expect(t, 0, len(w.Bytes()))
}

func prepareBenchmark(size, chunk, align int) ([]byte, []uint, []uint64, int) {
	buf := make([]byte, size)
	bits := make([]uint, size)
	values := make([]uint64, size)
	idx := 0
	last := 0
	for i := 0; i < size; i++ {
		val := getNumBits(idx, size*8, chunk, align)
		idx += val
		if val != 0 {
			last = i + 1
		}
		bits[i] = uint(val)
		values[i] = uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
	}
	return buf, bits, values, last
}

func bigWrite32(w *Writer, bits []uint, values []uint64, last int) {
	for j := 0; j < last; j++ {
		w.PutUint32Be(bits[j], uint32(values[j]))
	}
}

func bigWrite64(w *Writer, bits []uint, values []uint64, last int) {
	for j := 0; j < last; j++ {
		w.PutUint64Be(bits[j], values[j])
	}
}

func littleWrite32(w *Writer, bits []uint, values []uint64, last int) {
	for j := 0; j < last; j++ {
		w.PutUint32Le(bits[j], uint32(values[j]))
	}
}

func littleWrite64(w *Writer, bits []uint, values []uint64, last int) {
	for j := 0; j < last; j++ {
		w.PutUint64Le(bits[j], values[j])
	}
}

type WriteOp func(*Writer, []uint, []uint64, int)

func benchmarkWrites(b *testing.B, op WriteOp, chunk, align int) {
	size := 1 << 12
	buf, bits, values, last := prepareBenchmark(size, chunk, align)
	b.SetBytes(int64(len(buf)))
	w := NewWriter(buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Reset()
		op(w, bits, values, last)
	}
}

func BenchmarkBigEndianWriteUint32Align1(b *testing.B)     { benchmarkWrites(b, bigWrite32, 32, 1) }
func BenchmarkBigEndianWriteUint32Align32(b *testing.B)    { benchmarkWrites(b, bigWrite32, 32, 32) }
func BenchmarkBigEndianWriteUint64Align1(b *testing.B)     { benchmarkWrites(b, bigWrite64, 64, 1) }
func BenchmarkBigEndianWriteUint64Align32(b *testing.B)    { benchmarkWrites(b, bigWrite64, 64, 32) }
func BenchmarkBigEndianWriteUint64Align64(b *testing.B)    { benchmarkWrites(b, bigWrite64, 64, 64) }
func BenchmarkLittleEndianWriteUint32Align1(b *testing.B)  { benchmarkWrites(b, littleWrite32, 32, 1) }
func BenchmarkLittleEndianWriteUint32Align32(b *testing.B) { benchmarkWrites(b, littleWrite32, 32, 32) }
func BenchmarkLittleEndianWriteUint64Align1(b *testing.B)  { benchmarkWrites(b, littleWrite64, 64, 1) }
func BenchmarkLittleEndianWriteUint64Align32(b *testing.B) { benchmarkWrites(b, littleWrite64, 64, 32) }
func BenchmarkLittleEndianWriteUint64Align64(b *testing.B) { benchmarkWrites(b, littleWrite64, 64, 64) }
