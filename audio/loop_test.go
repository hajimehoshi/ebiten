// Copyright 2018 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package audio_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

func TestInfiniteLoop(t *testing.T) {
	indexToByte := func(index int) byte {
		return byte(math.Sin(float64(index)) * 256)
	}

	src := make([]byte, 256)
	for i := range src {
		src[i] = indexToByte(i)
	}
	l := audio.NewInfiniteLoop(bytes.NewReader(src), int64(len(src)))

	buf := make([]byte, len(src)*4)
	if _, err := io.ReadFull(l, buf); err != nil {
		t.Error(err)
	}
	for i, b := range buf {
		got := b
		want := indexToByte(i % len(src))
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	n, err := l.Seek(int64(len(src))*5+128, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if want := int64(128); n != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	n2, err := l.Seek(int64(len(src))*6+64, io.SeekCurrent)
	if err != nil {
		t.Error(err)
	}
	if want := int64(192); n2 != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	buf2 := make([]byte, len(src)*7)
	if _, err := io.ReadFull(l, buf2); err != nil {
		t.Error(err)
	}
	for i, b := range buf2 {
		got := b
		want := indexToByte((i + 192) % len(src))
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	// Seek to negative position is an error.
	if _, err := l.Seek(-1, io.SeekStart); err == nil {
		t.Errorf("got: %v, want: %v", err, nil)
	}
}

func TestInfiniteLoopWithIntro(t *testing.T) {
	const (
		srcLength   = 17 * 4
		introLength = 19 * 4
		loopLength  = 23 * 4
	)

	indexToByte := func(index int) byte {
		return byte(math.Sin(float64(index)) * 256)
	}
	src := make([]byte, srcLength)
	for i := range src {
		src[i] = indexToByte(i)
	}
	srcInf := audio.NewInfiniteLoop(bytes.NewReader(src), srcLength)
	srcInf.SetNoBlendForTesting(true)
	l := audio.NewInfiniteLoopWithIntro(srcInf, introLength, loopLength)
	l.SetNoBlendForTesting(true)

	buf := make([]byte, srcLength*4)
	if _, err := io.ReadFull(l, buf); err != nil {
		t.Error(err)
	}
	for i, b := range buf {
		got := b
		var want byte
		if i < introLength {
			want = indexToByte(i % srcLength)
		} else {
			want = indexToByte(((i-introLength)%loopLength + introLength) % srcLength)
		}
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	n, err := l.Seek(srcLength*5+128, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if want := int64((srcLength*5+128-introLength)%loopLength + introLength); n != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	n2, err := l.Seek(srcLength*6+64, io.SeekCurrent)
	if err != nil {
		t.Error(err)
	}
	if want := int64(((srcLength*11+192)-introLength)%loopLength + introLength); n2 != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	buf2 := make([]byte, srcLength*7)
	if _, err := io.ReadFull(l, buf2); err != nil {
		t.Error(err)
	}
	for i, b := range buf2 {
		got := b
		idx := ((int(n2+int64(i))-introLength)%loopLength + introLength) % srcLength
		want := indexToByte(idx)
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	// Seek to negative position is an error.
	if _, err := l.Seek(-1, io.SeekStart); err == nil {
		t.Errorf("got: %v, want: %v", err, nil)
	}
}

func TestInfiniteLoopWithPartialFrameAfterLoop(t *testing.T) {
	src := bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9})
	l := audio.NewInfiniteLoop(src, 8)

	buf := make([]byte, 16)
	if _, err := l.Read(buf); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Read(buf); err != nil {
		t.Fatal(err)
	}
}

func TestInfiniteLoopWithIncompleteSize(t *testing.T) {
	// s1 should work as if 4092 is given.
	s1 := audio.NewInfiniteLoop(bytes.NewReader(make([]byte, 4096)), 4095)
	n1, err := s1.Seek(4094, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if got, want := n1, int64(4094-4092); got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}

	// s2 should work as if 2044 and 2044 are given.
	s2 := audio.NewInfiniteLoopWithIntro(bytes.NewReader(make([]byte, 4096)), 2047, 2046)
	n2, err := s2.Seek(4094, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if got, want := n2, int64(2044+(4094-(2044+2044))); got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}
}

type slowReader struct {
	src io.ReadSeeker
	eof bool
}

func (s *slowReader) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		if s.eof {
			return 0, io.EOF
		}
		return 0, nil
	}

	n, err := s.src.Read(buf[:1])
	if err == io.EOF {
		s.eof = true
	}
	return n, err
}

func (s *slowReader) Seek(offset int64, whence int) (int64, error) {
	s.eof = false
	return s.src.Seek(offset, whence)
}

// stalledReader delivers data until its source is exhausted, then reports the
// end as (0, nil) instead of io.EOF, like a real-time source that simply has
// no data ready yet. This is a legal io.Reader behavior.
type stalledReader struct {
	src io.ReadSeeker
}

func (s *stalledReader) Read(buf []byte) (int, error) {
	n, err := s.src.Read(buf)
	if err == io.EOF {
		return 0, nil
	}
	return n, err
}

func (s *stalledReader) Seek(offset int64, whence int) (int64, error) {
	return s.src.Seek(offset, whence)
}

func TestInfiniteLoopWithStalledSourceAfterLoop(t *testing.T) {
	const length = 4096
	src := make([]byte, length)
	loop := audio.NewInfiniteLoop(&stalledReader{src: bytes.NewReader(src)}, length)

	buf := make([]byte, length)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := loop.Read(buf); err != nil {
			t.Error(err)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Read hung: the after-loop read did not break on a (0, nil) source")
	}
}

func TestInfiniteLoopWithSlowSource(t *testing.T) {
	src := make([]byte, 4096)
	for i := range src {
		src[i] = byte(i)
	}
	r := &slowReader{
		src: bytes.NewReader(src),
	}
	loop := audio.NewInfiniteLoop(r, 4096)

	buf := make([]byte, 4096)

	// With a slow source, whose Read always reads at most one byte,
	// an infinite loop should return whole samples (bitDepthInBytes = 2) instead of no bytes.

	for i := range 4 {
		n, err := loop.Read(buf)
		if err != nil {
			t.Error(err)
		}
		if got, want := n, 2; got != want {
			t.Errorf("got: %d, want: %d", got, want)
		}
		if got, want := buf[0], byte(2*i); got != want {
			t.Errorf("got: %d, want: %d", got, want)
		}
		if got, want := buf[1], byte(2*i+1); got != want {
			t.Errorf("got: %d, want: %d", got, want)
		}
	}
}

func TestInfiniteLoopWithSlowSourceKeepsAllBytes(t *testing.T) {
	cases := []struct {
		name    string
		lstart  int
		length  int
		newLoop func(src io.ReadSeeker) *audio.InfiniteLoop
	}{
		{
			name:   "int16",
			lstart: 0,
			length: 16,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoop(src, 16)
			},
		},
		{
			name:   "int16 with intro",
			lstart: 8,
			length: 24,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoopWithIntro(src, 8, 16)
			},
		},
		{
			name:   "float32",
			lstart: 0,
			length: 32,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoopF32(src, 32)
			},
		},
		{
			name:   "float32 with intro",
			lstart: 8,
			length: 40,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoopWithIntroF32(src, 8, 32)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := make([]byte, c.length)
			for i := range src {
				src[i] = byte(i)
			}
			l := c.newLoop(&slowReader{
				src: bytes.NewReader(src),
			})

			// The source returns at most one byte per Read, so a remainder is carried over between
			// Reads. The remainder must not be counted twice in the loop position.
			buf := make([]byte, c.length)
			var out []byte
			for range c.length * 4 {
				n, err := l.Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				out = append(out, buf[:n]...)
			}

			if got, want := len(out), c.length*2; got < want {
				t.Errorf("len(out): %d, want >= %d", got, want)
			}
			for i, got := range out {
				idx := i
				if idx >= c.length {
					idx = (idx-c.lstart)%(c.length-c.lstart) + c.lstart
				}
				if want := src[idx]; got != want {
					t.Errorf("index: %d, got: %v, want: %v", i, got, want)
					break
				}
			}
		})
	}
}

// partialFrameReader is a reader whose Read returns at most size bytes.
type partialFrameReader struct {
	src  io.ReadSeeker
	size int
}

func (p *partialFrameReader) Read(buf []byte) (int, error) {
	if len(buf) > p.size {
		buf = buf[:p.size]
	}
	return p.src.Read(buf)
}

func (p *partialFrameReader) Seek(offset int64, whence int) (int64, error) {
	return p.src.Seek(offset, whence)
}

func TestInfiniteLoopBlendWithPartialFrameReads(t *testing.T) {
	cases := []struct {
		name    string
		length  int
		newLoop func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:    "int16",
			length:  2 * 2 * 2,
			newLoop: audio.NewInfiniteLoop,
		},
		{
			name:    "float32",
			length:  4 * 2 * 2,
			newLoop: audio.NewInfiniteLoopF32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The loop part is silent and the part after the loop is not, so the first sample of the
			// second lap, whose blend rate is 1, must be exactly the first sample after the loop.
			src := make([]byte, c.length*2)
			for i := c.length; i < len(src); i++ {
				src[i] = byte(100 + i - c.length)
			}
			// The source returns 3 bytes at most, which is not a multiple of any bit depth, so the
			// position of the blended data must not be affected by the remainder.
			l := c.newLoop(&partialFrameReader{
				src:  bytes.NewReader(src),
				size: 3,
			}, int64(c.length))

			// bytesPerSample is the size of one sample, which is the size of the data whose blend rate is 1.
			bytesPerSample := c.length / 2

			buf := make([]byte, c.length)
			var out []byte
			for range c.length * 2 {
				n, err := l.Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				out = append(out, buf[:n]...)
				if len(out) >= c.length+bytesPerSample {
					break
				}
			}
			if got, want := len(out), c.length+bytesPerSample; got < want {
				t.Fatalf("len(out): %d, want >= %d", got, want)
			}

			// The first lap is not blended as the data after the loop is not read yet.
			for i, got := range out[:c.length] {
				if want := byte(0); got != want {
					t.Errorf("index: %d, got: %v, want: %v", i, got, want)
					break
				}
			}
			if got, want := out[c.length:c.length+bytesPerSample], src[c.length:c.length+bytesPerSample]; !bytes.Equal(got, want) {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

func TestInfiniteLoopSeekClearsExtra(t *testing.T) {
	// The source returns 5 bytes at most, which is larger than any bit depth but not a multiple of
	// any, so a read returns a complete value and leaves a remainder.
	src := &partialFrameReader{
		src:  bytes.NewReader(bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 3)),
		size: 5,
	}
	l := audio.NewInfiniteLoopF32(src, 8)

	buf := make([]byte, 32)
	if _, err := l.Read(buf); err != nil {
		t.Fatal(err)
	}

	if _, err := l.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	n, err := l.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 2, 3, 4}
	if !bytes.Equal(buf[:n], want) {
		t.Errorf("got: %v, want: %v", buf[:n], want)
	}
}

func TestInfiniteLoopSeekCurrentAfterPartialFrameRead(t *testing.T) {
	// The source returns 5 bytes at most, which is larger than any bit depth but not a multiple of
	// any, so a read returns a complete value and leaves a remainder.
	src := &partialFrameReader{
		src:  bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		size: 5,
	}
	l := audio.NewInfiniteLoopF32(src, 8)

	buf := make([]byte, 32)
	// This read leaves a one-byte remainder in the internal buffer, so the source position is ahead of
	// the logical position by 1.
	if _, err := l.Read(buf); err != nil {
		t.Fatal(err)
	}

	// Seek with io.SeekCurrent must be relative to the logical position, so this must be a no-op.
	pos, err := l.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(4); pos != want {
		t.Errorf("got: %d, want: %d", pos, want)
	}

	n, err := l.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{5, 6, 7, 8}
	if !bytes.Equal(buf[:n], want) {
		t.Errorf("got: %v, want: %v", buf[:n], want)
	}
}

func TestInfiniteLoopZeroLoopLength(t *testing.T) {
	for _, tc := range []struct {
		name        string
		introLength int64
		loopLength  int64
	}{
		{"ShortLoop", 8, 3},
		{"ZeroLoop", 8, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected a panic for a zero loop length, but got none")
				}
			}()
			audio.NewInfiniteLoopWithIntro(bytes.NewReader(make([]byte, 64)), tc.introLength, tc.loopLength)
		})
	}
}

func TestInfiniteLoopSeekInvalidWhence(t *testing.T) {
	src := make([]byte, 256)
	l := audio.NewInfiniteLoop(bytes.NewReader(src), int64(len(src)))

	for _, whence := range []int{io.SeekEnd, -1, 3, 100} {
		if _, err := l.Seek(0, whence); err == nil {
			t.Errorf("Seek(0, %d): got no error, want an error", whence)
		}
	}
}

func TestInfiniteLoopShortBuffer(t *testing.T) {
	const srcLen = 4096

	src := make([]byte, srcLen)
	for i := range src {
		src[i] = byte(i)
	}

	cases := []struct {
		name    string
		newLoop func(src io.ReadSeeker) *audio.InfiniteLoop
		lens    []int
	}{
		{
			name: "int16",
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoop(src, srcLen)
			},
			lens: []int{1},
		},
		{
			name: "float32",
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoopF32(src, srcLen)
			},
			lens: []int{1, 2, 3},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loop := c.newLoop(bytes.NewReader(src))

			if n, err := loop.Read(nil); n != 0 || err != nil {
				t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
			}
			for _, l := range c.lens {
				if n, err := loop.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
					t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
				}
			}
		})
	}
}

func TestInfiniteLoopSeekAlignment(t *testing.T) {
	// A seek must land on a value boundary, so that the values read afterwards are the source's and
	// not ones straddling two of them.
	cases := []struct {
		name            string
		bitDepthInBytes int64
		newLoop         func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:            "int16",
			bitDepthInBytes: 2,
			newLoop:         audio.NewInfiniteLoop,
		},
		{
			name:            "float32",
			bitDepthInBytes: 4,
			newLoop:         audio.NewInfiniteLoopF32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := make([]byte, 64)
			for i := range src {
				src[i] = byte(i + 1)
			}

			for offset := range int64(16) {
				want := offset / c.bitDepthInBytes * c.bitDepthInBytes

				for _, whence := range []int{io.SeekStart, io.SeekCurrent} {
					l := c.newLoop(bytes.NewReader(src), int64(len(src)))
					pos, err := l.Seek(offset, whence)
					if err != nil {
						t.Errorf("Seek(%d, %d): %v", offset, whence, err)
						continue
					}
					if pos != want {
						t.Errorf("Seek(%d, %d): got %d, want %d", offset, whence, pos, want)
						continue
					}

					buf := make([]byte, 16)
					n, err := l.Read(buf)
					if err != nil {
						t.Errorf("Read after Seek(%d, %d): %v", offset, whence, err)
						continue
					}
					if got, w := buf[:n], src[want:want+int64(n)]; !bytes.Equal(got, w) {
						t.Errorf("Read after Seek(%d, %d): got %v, want %v", offset, whence, got, w)
					}
				}
			}
		})
	}
}
