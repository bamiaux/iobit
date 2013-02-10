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

func testWrites(t *testing.T, align int) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	src := makeSource(512)
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
	compare(t, src, buf.Bytes())
}

func TestWrites(t *testing.T) {
	for i := 32; i > 0; i >>= 1 {
		testWrites(t, i)
	}
}

func TestLittleEndian(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	LittleEndian.PutUint64(w, 64, 0x0123456789ABCDEF)
	w.Flush()
	compare(t, buf.Bytes(), []uint8{0xEF, 0xCD, 0xAB, 0x89, 0x67, 0x45, 0x23, 0x01})
}

func benchWrites(b *testing.B, align int) {
	b.StopTimer()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for i := 0; i < b.N; i++ {
		bits := uint(getNumBits(0, 1024, align))
		value := rand.Uint32()
		b.StartTimer()
		BigEndian.PutUint32(w, bits, value)
		b.StopTimer()
		buf.Reset()
	}
}

func BenchmarkWrites(b *testing.B) {
	benchWrites(b, 1)
}

func TestFlushOverflow(t *testing.T) {
	for i := 0; i < CacheSize*2; i++ {
		var buf bytes.Buffer
		w := NewWriterSize(&buf, CacheSize)
		for j := 0; j < i; j++ {
			BigEndian.PutUint32(w, 8, 0)
		}
		flushCheck(t, w)
	}
}

func TestSmallWriter(t *testing.T) {
	for i := CacheSize; i >= 0; i-- {
		var buf bytes.Buffer
		w := NewWriterSize(&buf, i)
		BigEndian.PutUint64(w, 64, 0)
		BigEndian.PutUint64(w, 64, 0)
		flushCheck(t, w)
	}
}
