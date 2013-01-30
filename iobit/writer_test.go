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

func MakeSource(size int) []uint8 {
	src := make([]uint8, size)
	for i := range src {
		src[i] = uint8(rand.Intn(0xFF))
	}
	return src[:]
}

func testWrites(t *testing.T, align int) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	src := MakeSource(512)
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
		w.WriteBits(uint(bits), value)
		read += bits
	}
	err := w.Flush()
	if err != nil {
		t.Fatal("error during write", err)
	}
	if dst := buf.Bytes(); !bytes.Equal(src, dst) {
		t.Log(hex.Dump(src))
		t.Log(hex.Dump(dst))
		t.Fatal("invalid output")
	}
}

func TestWrites(t *testing.T) {
	for i := 32; i > 0; i >>= 1 {
		testWrites(t, i)
	}
}

func benchWrites(b *testing.B, align int) {
	b.StopTimer()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for i := 0; i < b.N; i++ {
		bits := uint(getNumBits(0, 1024, align))
		value := rand.Uint32()
		b.StartTimer()
		w.WriteBits(bits, value)
		b.StopTimer()
		buf.Reset()
	}
}

func BenchmarkWrites(b *testing.B) {
	benchWrites(b, 1)
}
