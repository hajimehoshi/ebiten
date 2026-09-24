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
	"encoding/binary"
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
	n1, err := s1.Seek(4096, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if got, want := n1, int64(4096-4092); got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}

	// s2 should work as if 2044 and 2044 are given.
	s2 := audio.NewInfiniteLoopWithIntro(bytes.NewReader(make([]byte, 4096)), 2047, 2046)
	n2, err := s2.Seek(4096, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if got, want := n2, int64(2044+(4096-(2044+2044))); got != want {
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

	for i := range 4 {
		n, err := loop.Read(buf)
		if err != nil {
			t.Error(err)
		}
		if got, want := n, 4; got != want {
			t.Errorf("got: %d, want: %d", got, want)
		}
		if got, want := buf[:4], []byte{byte(4 * i), byte(4*i + 1), byte(4*i + 2), byte(4*i + 3)}; !bytes.Equal(got, want) {
			t.Errorf("got: %v, want: %v", got, want)
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

// emptyOnceReader reports (0, nil) for the first read at or after afterPos, like a source whose
// data after the loop is not ready yet, and delegates to the source afterwards.
type emptyOnceReader struct {
	src      io.ReadSeeker
	afterPos int64
	emptied  bool
}

func (r *emptyOnceReader) Read(buf []byte) (int, error) {
	if !r.emptied && len(buf) > 0 {
		pos, err := r.src.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		if pos >= r.afterPos {
			r.emptied = true
			return 0, nil
		}
	}
	return r.src.Read(buf)
}

func (r *emptyOnceReader) Seek(offset int64, whence int) (int64, error) {
	return r.src.Seek(offset, whence)
}

func TestInfiniteLoopBlendAfterEmptyAfterLoopRead(t *testing.T) {
	const (
		length         = 16
		bytesPerSample = 4
	)

	// The loop part is silent and the part after the loop is not, so the first sample of a blended
	// lap, whose blend rate is 1, must be exactly the first sample after the loop.
	src := make([]byte, length*2)
	for i := length; i < len(src); i++ {
		src[i] = byte(100 + i - length)
	}
	l := audio.NewInfiniteLoop(&emptyOnceReader{
		src:      bytes.NewReader(src),
		afterPos: length,
	}, length)

	buf := make([]byte, length)
	var out []byte
	for len(out) < length*2+bytesPerSample {
		n, err := l.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, buf[:n]...)
	}

	// The first two laps are not blended: the data after the loop is captured only when the loop
	// end is reached again after the empty read.
	for i, got := range out[:length*2] {
		if want := byte(0); got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
			break
		}
	}
	if got, want := out[length*2:length*2+bytesPerSample], src[length:length+bytesPerSample]; !bytes.Equal(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestInfiniteLoopKeepsBlendingOnFailedSeek(t *testing.T) {
	cases := []struct {
		name           string
		bytesPerSample int
		length         int
		newLoop        func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:           "int16",
			bytesPerSample: 2 * 2,
			length:         2 * 2 * 4,
			newLoop:        audio.NewInfiniteLoop,
		},
		{
			name:           "float32",
			bytesPerSample: 4 * 2,
			length:         4 * 2 * 4,
			newLoop:        audio.NewInfiniteLoopF32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The loop part is silent and the part after the loop is not, so a blended sample at the
			// loop start, whose blend rate is 1, differs from the raw silent loop data.
			src := make([]byte, c.length*2)
			for i := c.length; i < len(src); i++ {
				src[i] = byte(100 + i - c.length)
			}
			l := c.newLoop(bytes.NewReader(src), int64(c.length))

			// Read through the loop boundary so that the data after the loop is read and blending
			// becomes active.
			buf := make([]byte, c.length+c.bytesPerSample)
			firstLap := make([]byte, 0, c.length)
			for len(firstLap) < c.length {
				n, err := l.Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				firstLap = append(firstLap, buf[:n]...)
			}
			// The first lap is not blended as the data after the loop was not read yet.
			if got, want := firstLap[:c.length], src[:c.length]; !bytes.Equal(got, want) {
				t.Fatalf("first lap: got: %v, want: %v", got, want)
			}

			// Seeking to a negative position fails, and the loop is left at the loop start.
			if _, err := l.Seek(-1, io.SeekStart); err == nil {
				t.Fatal("Seek(-1, io.SeekStart): got no error, want an error")
			}

			// The next samples must still be blended: the first sample at the loop start, whose blend
			// rate is 1, must be the first sample after the loop rather than the raw silent loop start.
			if _, err := l.Read(buf); err != nil {
				t.Fatal(err)
			}
			if got, want := buf[:c.bytesPerSample], src[c.length:c.length+c.bytesPerSample]; !bytes.Equal(got, want) {
				t.Errorf("blending after a failed seek: got: %v, want: %v", got, want)
			}
		})
	}
}

func TestInfiniteLoopSeekClearsExtra(t *testing.T) {
	src := &partialFrameReader{
		src:  bytes.NewReader(bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 3)),
		size: 5,
	}
	l := audio.NewInfiniteLoopF32(src, 16)

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
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	if !bytes.Equal(buf[:n], want) {
		t.Errorf("got: %v, want: %v", buf[:n], want)
	}
}

func TestInfiniteLoopSeekCurrentAfterPartialFrameRead(t *testing.T) {
	data := make([]byte, 24)
	for i := range data {
		data[i] = byte(i + 1)
	}
	src := &partialFrameReader{
		src:  bytes.NewReader(data),
		size: 5,
	}
	l := audio.NewInfiniteLoopF32(src, int64(len(data)))

	n, err := l.Read(make([]byte, 9))
	if err != nil {
		t.Fatal(err)
	}

	pos, err := l.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(n); pos != want {
		t.Errorf("got: %d, want: %d", pos, want)
	}

	got := make([]byte, 8)
	if _, err := io.ReadFull(l, got); err != nil {
		t.Fatal(err)
	}
	if want := data[n : n+len(got)]; !bytes.Equal(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
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
	// A seek must land on a sample boundary, so that the samples read afterwards are the source's and
	// not ones straddling two of them.
	cases := []struct {
		name           string
		bytesPerSample int64
		newLoop        func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:           "int16",
			bytesPerSample: 4,
			newLoop:        audio.NewInfiniteLoop,
		},
		{
			name:           "float32",
			bytesPerSample: 8,
			newLoop:        audio.NewInfiniteLoopF32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := make([]byte, 64)
			for i := range src {
				src[i] = byte(i + 1)
			}

			for offset := range int64(16) {
				want := offset / c.bytesPerSample * c.bytesPerSample

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

func TestInfiniteLoopWithSourceEndingBeforeLoop(t *testing.T) {
	for _, tc := range []struct {
		name        string
		srcLength   int
		introLength int64
	}{
		{
			name:        "ShorterThanIntro",
			srcLength:   64,
			introLength: 256,
		},
		{
			name:        "EmptySource",
			srcLength:   0,
			introLength: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := audio.NewInfiniteLoopWithIntro(bytes.NewReader(make([]byte, tc.srcLength)), tc.introLength, 1024)

			buf := make([]byte, 32)
			var err error
			for range 16 {
				var n int
				n, err = l.Read(buf)
				if err != nil {
					break
				}
				if n == 0 {
					t.Fatal("Read returned (0, nil) for a source ending before the loop, want io.EOF")
				}
			}
			if !errors.Is(err, io.EOF) {
				t.Errorf("got: %v, want: %v", err, io.EOF)
			}
		})
	}
}

func TestInfiniteLoopWithSourceEndingInsideLoop(t *testing.T) {
	src := make([]byte, 64)
	for i := range src {
		src[i] = byte(i + 1)
	}
	l := audio.NewInfiniteLoop(bytes.NewReader(src), 1024)

	// The source ends before the specified length. The loop is shortened to end there, and Read
	// never returns (0, nil).
	buf := make([]byte, 32)
	for i := range 8 {
		n, err := l.Read(buf)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if n == 0 {
			t.Fatal("Read returned (0, nil)")
		}
		pos := (i * len(buf)) % len(src)
		if got, want := buf[:n], src[pos:pos+n]; !bytes.Equal(got, want) {
			t.Errorf("Read %d: got: %v, want: %v", i, got, want)
		}
	}
}

func TestInfiniteLoopWithPartialValueAtLoopStart(t *testing.T) {
	for _, tc := range []struct {
		name      string
		srcLength int
		newLoop   func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:      "int16",
			srcLength: 1,
			newLoop:   audio.NewInfiniteLoop,
		},
		{
			name:      "float32",
			srcLength: 3,
			newLoop:   audio.NewInfiniteLoopF32,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The source is shorter than one value, so a read at the loop start never completes a
			// value by itself.
			l := tc.newLoop(bytes.NewReader(make([]byte, tc.srcLength)), 1024)

			buf := make([]byte, 32)
			done := make(chan struct{})
			go func() {
				defer close(done)
				if _, err := l.Read(buf); err != nil && !errors.Is(err, io.EOF) {
					t.Error(err)
				}
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Read hung: the retries at the loop start did not end")
			}
		})
	}
}

func TestInfiniteLoopRewindClearsExtra(t *testing.T) {
	src := make([]byte, 101)
	for i := range src {
		src[i] = byte(i)
	}
	l := audio.NewInfiniteLoop(bytes.NewReader(src), 200)

	buf := make([]byte, 128)
	if _, err := l.Read(buf); err != nil {
		t.Fatal(err)
	}
	n, err := l.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := buf[:4], []byte{0, 1, 2, 3}; !bytes.Equal(got[:4], want) {
		t.Errorf("got: %v, want: %v (n: %d)", got, want, n)
	}
}

// dataWithErrorReadSeeker returns at most dataN bytes together with errSourceFailed on the Read
// call with the given 1-based number.
type dataWithErrorReadSeeker struct {
	src    io.ReadSeeker
	failAt int
	dataN  int
	reads  int
}

func (d *dataWithErrorReadSeeker) Read(buf []byte) (int, error) {
	d.reads++
	if d.reads != d.failAt {
		return d.src.Read(buf)
	}
	n, err := d.src.Read(buf[:min(len(buf), d.dataN)])
	if err != nil {
		return n, err
	}
	return n, errSourceFailed
}

func (d *dataWithErrorReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return d.src.Seek(offset, whence)
}

func TestInfiniteLoopSourceErrorWithData(t *testing.T) {
	cases := []struct {
		name           string
		bytesPerSample int
		newLoop        func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:           "int16",
			bytesPerSample: 2 * 2,
			newLoop:        audio.NewInfiniteLoop,
		},
		{
			name:           "float32",
			bytesPerSample: 4 * 2,
			newLoop:        audio.NewInfiniteLoopF32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			length := 16 * c.bytesPerSample
			src := make([]byte, 2*length)
			for i := range src {
				src[i] = byte(i + 1)
			}
			want := make([]byte, 3*length)
			if _, err := io.ReadFull(c.newLoop(bytes.NewReader(src), int64(length)), want); err != nil {
				t.Fatal(err)
			}

			l := c.newLoop(&dataWithErrorReadSeeker{
				src:    bytes.NewReader(src),
				failAt: 1,
				dataN:  2*c.bytesPerSample + 1,
			}, int64(length))

			buf := make([]byte, length/2)
			n, err := l.Read(buf)
			if !errors.Is(err, errSourceFailed) {
				t.Errorf("Read: got error %v, want %v", err, errSourceFailed)
			}
			if got, want := n, 2*c.bytesPerSample; got != want {
				t.Errorf("Read: got %d bytes, want %d", got, want)
			}
			if got, want := buf[:n], want[:n]; !bytes.Equal(got, want) {
				t.Errorf("Read: got %v, want %v", got, want)
			}

			pos, err := l.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, int64(n); got != want {
				t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", got, want)
			}

			rest := make([]byte, len(want)-n)
			if _, err := io.ReadFull(l, rest); err != nil {
				t.Fatal(err)
			}
			if got, want := rest, want[n:]; !bytes.Equal(got, want) {
				t.Errorf("reading on after the error: got %v, want %v", got, want)
			}
		})
	}
}

func TestInfiniteLoopAfterLoopSourceErrorWithData(t *testing.T) {
	cases := []struct {
		name           string
		bytesPerSample int
		newLoop        func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:           "int16",
			bytesPerSample: 2 * 2,
			newLoop:        audio.NewInfiniteLoop,
		},
		{
			name:           "float32",
			bytesPerSample: 4 * 2,
			newLoop:        audio.NewInfiniteLoopF32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			length := 16 * c.bytesPerSample
			src := make([]byte, 2*length)
			for i := length; i < len(src); i++ {
				src[i] = byte(100 + i - length)
			}
			want := make([]byte, 3*length)
			if _, err := io.ReadFull(c.newLoop(bytes.NewReader(src), int64(length)), want); err != nil {
				t.Fatal(err)
			}

			l := c.newLoop(&dataWithErrorReadSeeker{
				src:    bytes.NewReader(src),
				failAt: 2,
				dataN:  c.bytesPerSample + 1,
			}, int64(length))

			buf := make([]byte, length)
			n, err := l.Read(buf)
			if !errors.Is(err, errSourceFailed) {
				t.Errorf("Read: got error %v, want %v", err, errSourceFailed)
			}
			if got, want := n, length; got != want {
				t.Errorf("Read: got %d bytes, want %d", got, want)
			}
			if got, want := buf[:n], want[:n]; !bytes.Equal(got, want) {
				t.Errorf("Read: got %v, want %v", got, want)
			}

			rest := make([]byte, len(want)-n)
			if _, err := io.ReadFull(l, rest); err != nil {
				t.Fatal(err)
			}
			if got, want := rest, want[n:]; !bytes.Equal(got, want) {
				t.Errorf("reading on after the error: got %v, want %v", got, want)
			}
		})
	}
}

func TestInfiniteLoopShortBufferEOFAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		depth   int
		newLoop func(io.ReadSeeker, int64, int64) *audio.InfiniteLoop
	}{
		{
			name:    "Int16",
			depth:   2,
			newLoop: audio.NewInfiniteLoopWithIntro,
		},
		{
			name:    "Float32",
			depth:   4,
			newLoop: audio.NewInfiniteLoopWithIntroF32,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, intro := range []int64{0, 16} {
				for size := 1; size < tc.depth; size++ {
					l := tc.newLoop(bytes.NewReader(nil), intro, 16)
					for range 2 {
						if n, err := l.Read(make([]byte, size)); n != 0 || !errors.Is(err, io.EOF) {
							t.Errorf("short Read from empty loop = (%d, %v)", n, err)
						}
					}
				}
			}
			src := make([]byte, 128)
			for i := range src {
				src[i] = byte(i)
			}
			want := make([]byte, 256)
			if _, err := io.ReadFull(tc.newLoop(bytes.NewReader(src), 16, 32), want); err != nil {
				t.Fatal(err)
			}
			for _, fail := range []bool{false, true} {
				var source io.ReadSeeker = bytes.NewReader(src)
				wantErr := io.ErrShortBuffer
				if fail {
					source = &dataWithErrorReadSeeker{
						src:    source,
						failAt: 1,
						dataN:  1,
					}
					wantErr = errSourceFailed
				}
				l := tc.newLoop(source, 16, 32)
				if n, err := l.Read(make([]byte, 1)); n != 0 || !errors.Is(err, wantErr) {
					t.Errorf("short Read = (%d, %v), want %v", n, err, wantErr)
				}
				got := make([]byte, len(want))
				if _, err := io.ReadFull(l, got); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Error("loop audio changed after short Read")
				}
			}
			l := tc.newLoop(bytes.NewReader(src), 16, 32)
			if _, err := l.Read(make([]byte, 1)); !errors.Is(err, io.ErrShortBuffer) {
				t.Fatal(err)
			}
			if pos, err := l.Seek(0, io.SeekCurrent); err != nil || pos != 0 {
				t.Errorf("current after short Read = (%d, %v), want 0", pos, err)
			}
		})
	}
}

// loopSourceWithAfterLoop returns a source whose loop part of length bytes is silent and which is
// followed by afterLoopSamples non-silent samples, encoded for the bit depth.
func loopSourceWithAfterLoop(length int, bitDepthInBytes int, afterLoopSamples int) []byte {
	src := make([]byte, length)
	for range afterLoopSamples * 2 {
		switch bitDepthInBytes {
		case 2:
			src = binary.LittleEndian.AppendUint16(src, 1600)
		case 4:
			bits := math.Float32bits(1)
			src = append(src, byte(bits), byte(bits>>8), byte(bits>>16), byte(bits>>24))
		}
	}
	return src
}

// failingSeekReader is a source whose Seek fails with err while fail is set.
type failingSeekReader struct {
	src  io.ReadSeeker
	fail bool
	err  error
}

func (f *failingSeekReader) Read(buf []byte) (int, error) {
	return f.src.Read(buf)
}

func (f *failingSeekReader) Seek(offset int64, whence int) (int64, error) {
	if f.fail {
		return 0, f.err
	}
	return f.src.Seek(offset, whence)
}

func TestInfiniteLoopKeepsBlendingOnFailedSourceSeek(t *testing.T) {
	cases := []struct {
		name            string
		bitDepthInBytes int
		newLoop         func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
		halfSample      []byte
	}{
		{
			name:            "int16",
			bitDepthInBytes: 2,
			newLoop:         audio.NewInfiniteLoop,
			halfSample:      binary.LittleEndian.AppendUint16(binary.LittleEndian.AppendUint16(nil, 800), 800),
		},
		{
			name:            "float32",
			bitDepthInBytes: 4,
			newLoop:         audio.NewInfiniteLoopF32,
			halfSample: func() []byte {
				bits := math.Float32bits(0.5)
				half := []byte{byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)}
				return append(half, half...)
			}(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			const length = 64
			bytesPerSample := c.bitDepthInBytes * 2
			src := loopSourceWithAfterLoop(length, c.bitDepthInBytes, 2)
			seekErr := errors.New("audio_test: seek failed")
			r := &failingSeekReader{
				src: bytes.NewReader(src),
				err: seekErr,
			}
			l := c.newLoop(r, length)

			if _, err := io.ReadFull(l, make([]byte, length)); err != nil {
				t.Fatal(err)
			}
			first := make([]byte, bytesPerSample)
			if _, err := io.ReadFull(l, first); err != nil {
				t.Fatal(err)
			}
			if got, want := first, src[length:length+bytesPerSample]; !bytes.Equal(got, want) {
				t.Fatalf("the first sample of the second lap: got %v, want %v (blended with the data after the loop)", got, want)
			}

			r.fail = true
			if _, err := l.Seek(0, io.SeekStart); !errors.Is(err, seekErr) {
				t.Fatalf("Seek: got %v, want %v", err, seekErr)
			}
			r.fail = false

			second := make([]byte, bytesPerSample)
			if _, err := io.ReadFull(l, second); err != nil {
				t.Fatal(err)
			}
			if got, want := second, c.halfSample; !bytes.Equal(got, want) {
				t.Errorf("the second sample of the second lap after a failed Seek: got %v, want %v (still blended)", got, want)
			}
		})
	}
}

func TestInfiniteLoopReadsWholeSamples(t *testing.T) {
	cases := []struct {
		name           string
		bytesPerSample int
		newLoop        func(src io.ReadSeeker) *audio.InfiniteLoop
	}{
		{
			name:           "int16",
			bytesPerSample: 4,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoop(src, 64)
			},
		},
		{
			name:           "int16 with intro",
			bytesPerSample: 4,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoopWithIntro(src, 24, 40)
			},
		},
		{
			name:           "float32",
			bytesPerSample: 8,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoopF32(src, 64)
			},
		},
		{
			name:           "float32 with intro",
			bytesPerSample: 8,
			newLoop: func(src io.ReadSeeker) *audio.InfiniteLoop {
				return audio.NewInfiniteLoopWithIntroF32(src, 24, 40)
			},
		},
	}

	const limit = 1024
	sizes := []int{1, 5, 3, 6, 2, 7, 9, 13, 4, 10, 8, 27}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := make([]byte, 96)
			for i := range src {
				src[i] = byte(i*7 + 1)
			}
			want := make([]byte, limit)
			if _, err := io.ReadFull(c.newLoop(bytes.NewReader(src)), want); err != nil {
				t.Fatal(err)
			}

			for _, slow := range []bool{false, true} {
				var r io.ReadSeeker = bytes.NewReader(src)
				if slow {
					r = &slowReader{
						src: r,
					}
				}
				l := c.newLoop(r)
				var got []byte
				for i := 0; len(got) < limit; i++ {
					size := sizes[i%len(sizes)]
					buf := make([]byte, size)
					n, err := l.Read(buf)
					if n%c.bytesPerSample != 0 {
						t.Errorf("slow=%t: Read(a buffer of %d bytes) at %d: got %d bytes, want a multiple of %d", slow, size, len(got), n, c.bytesPerSample)
					}
					got = append(got, buf[:n]...)
					if size < c.bytesPerSample && errors.Is(err, io.ErrShortBuffer) {
						continue
					}
					if err != nil {
						t.Fatalf("slow=%t: Read(a buffer of %d bytes) at %d: %v", slow, size, len(got), err)
					}
				}
				if !bytes.Equal(got[:limit], want) {
					t.Errorf("slow=%t: the data differs from reading with aligned buffers", slow)
				}
			}
		})
	}
}

// wholeSampleReader returns whole samples only, and io.ErrShortBuffer for a buffer shorter than a sample.
type wholeSampleReader struct {
	src            *bytes.Reader
	bytesPerSample int
}

func (w *wholeSampleReader) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	if len(buf) < w.bytesPerSample {
		if w.src.Len() == 0 {
			return 0, io.EOF
		}
		return 0, io.ErrShortBuffer
	}
	return w.src.Read(buf[:len(buf)/w.bytesPerSample*w.bytesPerSample])
}

func (w *wholeSampleReader) Seek(offset int64, whence int) (int64, error) {
	return w.src.Seek(offset, whence)
}

func TestInfiniteLoopReadAfterShortBufferFromWholeSampleSource(t *testing.T) {
	cases := []struct {
		name           string
		bytesPerSample int
		newLoop        func(src io.ReadSeeker, length int64) *audio.InfiniteLoop
	}{
		{
			name:           "int16",
			bytesPerSample: 4,
			newLoop:        audio.NewInfiniteLoop,
		},
		{
			name:           "float32",
			bytesPerSample: 8,
			newLoop:        audio.NewInfiniteLoopF32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := make([]byte, 8*c.bytesPerSample)
			for i := range data {
				data[i] = byte(i + 1)
			}
			for size := c.bytesPerSample; size < 2*c.bytesPerSample; size++ {
				l := c.newLoop(&wholeSampleReader{
					src:            bytes.NewReader(data),
					bytesPerSample: c.bytesPerSample,
				}, int64(len(data)))
				if n, err := l.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
					t.Errorf("Read(a buffer of 1 byte): got (%d, %v), want (0, %v)", n, err, io.ErrShortBuffer)
				}
				buf := make([]byte, size)
				n, err := l.Read(buf)
				if err != nil {
					t.Errorf("Read(a buffer of %d bytes) after a short buffer: %v", size, err)
				}
				if got, want := buf[:n], data[:c.bytesPerSample]; !bytes.Equal(got, want) {
					t.Errorf("Read(a buffer of %d bytes) after a short buffer: got %v, want %v", size, got, want)
				}
			}
		})
	}
}
