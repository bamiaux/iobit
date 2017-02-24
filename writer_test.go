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
		w.PutUint32(uint(bits), value)
		read += bits
	}
	flushCheck(t, w)
}

func TestWrites(t *testing.T) {
	src := makeSource(512)
	dst := make([]byte, len(src))
	for i := 32; i > 0; i >>= 1 {
		w := NewWriter(dst)
		testWrites(&w, t, i, src)
		compare(t, src, dst)
	}
}

func TestSmall64BigEndianWrite(t *testing.T) {
	buf := make([]byte, 5)
	w := NewWriter(buf)
	w.PutUint64(33, 0xFFFFFFFE00000001)
	w.PutUint32(7, 0)
	w.Flush()
	compare(t, buf, []byte{0x00, 0x00, 0x00, 0x00, 0x80})
}

func TestBigEndianWrite(t *testing.T) {
	buf := make([]byte, 8)
	w := NewWriter(buf)
	w.PutUint64(64, 0x0123456789ABCDEF)
	w.Flush()
	compare(t, buf, []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF})
}

func TestLittleEndianWrite(t *testing.T) {
	buf := make([]byte, 8)
	w := NewWriter(buf)
	w.PutLe64(0x0123456789ABCDEF)
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
	w.PutUint32(9, 0)
	checkError(t, ErrUnderflow, w.Flush())

	w = NewWriter(buf)
	w.PutUint32(16, 0)
	checkError(t, nil, w.Flush())

	w = NewWriter(buf)
	w.PutUint32(17, 0)
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

func bswap64(v uint64) uint64 {
	return uint64(bswap32(uint32(v>>32))) | uint64(bswap32(uint32(v&0xFFFFFFFF)))<<32
}

func TestWriteHelpers(t *testing.T) {
	buf := []byte{0x00}
	w := NewWriter(buf[:])
	expect(t, int(8), w.Bits())
	w.PutUint32(1, 0)
	expect(t, int(1), w.Index())
	expect(t, int(7), w.Bits())
	w.PutUint32(1, 1)
	w.PutUint32(5, 0)
	w.PutUint32(1, 1)
	err := w.Flush()
	expect(t, buf, []byte{0x41})
	expect(t, int(8), w.Index())
	expect(t, int(0), w.Bits())
	expect(t, 0, len(w.Bytes()))
	expect(t, nil, err)
	w.PutUint32(1, 0)
	expect(t, int(9), w.Index())
	expect(t, int(0), w.Bits())
	expect(t, 0, len(w.Bytes()))
	expect(t, ErrOverflow, w.Flush())
	src := uint64(0x1234ABCDEF556789)
	dst := make([]byte, 64)
	w = NewWriter(dst)
	expectwrite := func(bits uint, swap bool) {
		for w.Index()&7 != 0 {
			w.PutUint32(1, 0)
		}
		r := NewReader(dst)
		expect(t, nil, w.Flush())
		w.Reset()
		expected := src << (64 - bits)
		if swap {
			expected = bswap64(expected) << (64 - bits)
		}
		expect(t, r.Uint64(bits)<<(64-bits), expected)
	}
	w.PutBit(src&1 != 0)
	expectwrite(1, false)
	w.PutByte(uint8(src & 0xFF))
	expectwrite(8, false)
	w.PutLe16(uint16(src & 0xFFFF))
	expectwrite(16, true)
	w.PutBe16(uint16(src & 0xFFFF))
	expectwrite(16, false)
	w.PutLe32(uint32(src & 0xFFFFFFFF))
	expectwrite(32, true)
	w.PutBe32(uint32(src & 0xFFFFFFFF))
	expectwrite(32, false)
	w.PutLe64(src)
	expectwrite(64, true)
	w.PutBe64(src)
	expectwrite(64, false)
	w.PutUint8(7, uint8(src&0xFF))
	expectwrite(7, false)
	w.PutInt8(7, int8(src&0xFF))
	expectwrite(7, false)
	w.PutUint16(15, uint16(src&0xFFFF))
	expectwrite(15, false)
	w.PutInt16(15, int16(src&0xFFFF))
	expectwrite(15, false)
	w.PutUint32(31, uint32(src&0xFFFFFFFF))
	expectwrite(31, false)
	w.PutInt32(31, int32(src&0xFFFFFFFF))
	expectwrite(31, false)
	w.PutUint64(64, src)
	expectwrite(64, false)
	w.PutInt64(64, int64(src))
	expectwrite(64, false)
}

func TestBadSlices(t *testing.T) {
	dst := []byte{0x00, 0x01, 0x02}
	w := NewWriter(dst[:])
	compare(t, w.Bytes(), dst[:])
	w.PutUint32(8, 0)
	compare(t, w.Bytes(), dst[1:])
	w.PutUint32(16, 0)
	expect(t, 0, len(w.Bytes()))
}

type WriteBench struct {
	name string
	bits int
	op   func(w *Writer, v uint64)
}

func BenchmarkWrites(b *testing.B) {
	size := 32
	dst := make([]byte, size*8)
	src := make([]uint64, size)
	for i := range src {
		src[i] = uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
	}
	w := NewWriter(dst)
	for _, v := range []WriteBench{
		{"bit", 1, func(w *Writer, v uint64) { w.PutBit(v != 0) }},
		{"byte", 8, func(w *Writer, v uint64) { w.PutByte(byte(v)) }},
		{"le16", 16, func(w *Writer, v uint64) { w.PutLe16(uint16(v)) }},
		{"be16", 16, func(w *Writer, v uint64) { w.PutBe16(uint16(v)) }},
		{"le32", 32, func(w *Writer, v uint64) { w.PutLe32(uint32(v)) }},
		{"be32", 32, func(w *Writer, v uint64) { w.PutBe32(uint32(v)) }},
		{"le64", 64, func(w *Writer, v uint64) { w.PutLe64(v) }},
		{"be64", 64, func(w *Writer, v uint64) { w.PutBe64(v) }},
		{"u8 7bits", 7, func(w *Writer, v uint64) { w.PutUint8(7, uint8(v)) }},
		{"i8 7bits", 7, func(w *Writer, v uint64) { w.PutInt8(7, int8(v)) }},
		{"u16 15bits", 15, func(w *Writer, v uint64) { w.PutUint16(15, uint16(v)) }},
		{"i16 15bits", 15, func(w *Writer, v uint64) { w.PutInt16(15, int16(v)) }},
		{"u32 31bits", 31, func(w *Writer, v uint64) { w.PutUint32(31, uint32(v)) }},
		{"i32 31bits", 31, func(w *Writer, v uint64) { w.PutInt32(31, int32(v)) }},
		{"u64 63bits", 63, func(w *Writer, v uint64) { w.PutUint64(63, v) }},
		{"i64 63bits", 63, func(w *Writer, v uint64) { w.PutInt64(63, int64(v)) }},
	} {
		b.Run(v.name, func(bb *testing.B) {
			bb.SetBytes(int64(v.bits * len(src)))
			for i := 0; i < bb.N; i++ {
				w.Reset()
				for _, k := range src {
					v.op(&w, k)
				}
			}
		})
	}
}
