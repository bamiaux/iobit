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
	dst := make([]uint8, len(src))
	for i := 32; i > 0; i >>= 1 {
		w := NewWriter(dst)
		testWrites(w, t, i, src)
		compare(t, src, dst)
	}
}

func TestSmall64BigEndianWrite(t *testing.T) {
	buf := make([]uint8, 5)
	w := NewWriter(buf)
	BigEndian.PutUint64(w, 33, 0xFFFFFFFE00000001)
	BigEndian.PutUint32(w, 7, 0)
	w.Flush()
	compare(t, buf, []uint8{0x00, 0x00, 0x00, 0x00, 0x80})
}

func TestSmall64LittleEndianWrite(t *testing.T) {
	buf := make([]uint8, 5)
	w := NewWriter(buf)
	LittleEndian.PutUint64(w, 33, 0xFFFFFFFE00000001)
	LittleEndian.PutUint32(w, 7, 0)
	w.Flush()
	compare(t, buf, []uint8{0x01, 0x00, 0x00, 0x00, 0x00})
}

func TestBigEndianWrite(t *testing.T) {
	buf := make([]uint8, 8)
	w := NewWriter(buf)
	BigEndian.PutUint64(w, 64, 0x0123456789ABCDEF)
	w.Flush()
	compare(t, buf, []uint8{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF})
}

func TestLittleEndianWrite(t *testing.T) {
	buf := make([]uint8, 8)
	w := NewWriter(buf)
	LittleEndian.PutUint64(w, 64, 0x0123456789ABCDEF)
	w.Flush()
	compare(t, buf, []uint8{0xEF, 0xCD, 0xAB, 0x89, 0x67, 0x45, 0x23, 0x01})
}

func benchmarkWrites(b *testing.B, align int) {
	b.StopTimer()
	size := 1 << 12
	buf := make([]uint8, size)
	bits := make([]uint, size)
	values := make([]uint32, size)
	idx := 0
	last := 0
	for i := 0; i < size; i++ {
		val := getNumBits(idx, size*8, align)
		idx += val
		if val != 0 {
			last = i + 1
		}
		bits[i] = uint(val)
		values[i] = rand.Uint32()
	}
	b.SetBytes(int64(size))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		w := NewWriter(buf)
		for j := 0; j < last; j++ {
			BigEndian.PutUint32(w, bits[j], values[j])
		}
	}
}

func BenchmarkBigEndian1(b *testing.B)  { benchmarkWrites(b, 1) }
func BenchmarkBigEndian8(b *testing.B)  { benchmarkWrites(b, 8) }
func BenchmarkBigEndian32(b *testing.B) { benchmarkWrites(b, 32) }

func checkError(t *testing.T, expected, actual error) {
	if actual != expected {
		t.Fatal("expecting", expected, "got", actual)
	}
}

func TestFlushErrors(t *testing.T) {
	buf := make([]uint8, 2)

	w := NewWriter(buf)
	BigEndian.PutUint32(w, 9, 0)
	checkError(t, ErrUnderflow, w.Flush())

	w = NewWriter(buf)
	BigEndian.PutUint32(w, 16, 0)
	checkError(t, nil, w.Flush())

	w = NewWriter(buf)
	BigEndian.PutUint32(w, 17, 0)
	checkError(t, ErrOverflow, w.Flush())
}
