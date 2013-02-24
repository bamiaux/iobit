package iobit

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"testing"
)

func getNumBits(read, max, align int) int {
	bits := 1
	if align != 32 {
		bits += rand.Intn(32 / align)
	}
	bits *= align
	if read+bits > max {
		bits = max - read
	}
	if bits > 32 {
		panic("too many bits")
	}
	return bits
}

func makeSource(size int) []uint8 {
	src := make([]uint8, size)
	for i := range src {
		src[i] = uint8(rand.Intn(0xFF))
	}
	return src[:]
}

func flushCheck(t *testing.T, w *Writer) {
	err := w.Flush()
	if err != nil {
		t.Fatal("unexpected error during flush", err)
	}
}

func compare(t *testing.T, src, dst []uint8) {
	if bytes.Equal(src, dst) {
		return
	}
	t.Log(hex.Dump(src))
	t.Log(hex.Dump(dst))
	t.Fatal("invalid output")
}

func testWrites(w *Writer, t *testing.T, align int, src []uint8) {
	max := len(src) * 8
	for read := 0; read < max; {
		bits := getNumBits(read, max, align)
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
		BigEndian.PutUint32(w, uint(bits), value)
		read += bits
	}
	flushCheck(t, w)
}

func TestWrites(t *testing.T) {
	src := makeSource(512)
	var buf bytes.Buffer
	for i := 32; i > 0; i >>= 1 {
		buf.Reset()
		w := NewWriter(&buf)
		testWrites(w, t, i, src)
		compare(t, src, buf.Bytes())
	}
	dst := make([]uint8, len(src))
	for i := 32; i > 0; i >>= 1 {
		w := NewRawWriter(dst)
		testWrites(w, t, i, src)
		compare(t, src, dst)
	}
}

func TestLittleEndian(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	LittleEndian.PutUint64(w, 64, 0x0123456789ABCDEF)
	w.Flush()
	compare(t, buf.Bytes(), []uint8{0xEF, 0xCD, 0xAB, 0x89, 0x67, 0x45, 0x23, 0x01})
}

func BenchmarkWrites(b *testing.B) {
	b.StopTimer()
	align := 1
	size := 1 << 12
	buf := make([]uint8, size)
	bits := make([]uint, size)
	values := make([]uint32, size)
	for i := 0; i < size; i++ {
		bits[i] = uint(getNumBits(0, size*8, align))
		values[i] = rand.Uint32()
	}
	b.SetBytes(int64(size))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		w := NewRawWriter(buf)
		for j := 0; j < size; j++ {
			BigEndian.PutUint32(w, bits[j], values[j])
		}
	}
}

func TestFlushOverflow(t *testing.T) {
	for i := 0; i < 64; i++ {
		var buf bytes.Buffer
		w := NewWriter(&buf)
		for j := 0; j < i; j++ {
			BigEndian.PutUint32(w, 8, 0)
		}
		flushCheck(t, w)
	}
}
