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

package convert_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

func soundAt(timeInSecond float64) float64 {
	const freq = 220

	amp := []float64{1.0, 0.8, 0.6, 0.4, 0.2}
	var v float64
	for j := range amp {
		v += amp[j] * math.Sin(2.0*math.Pi*timeInSecond*freq*float64(j+1)) / 2
	}
	if v > 1 {
		v = 1
	}
	if v < -1 {
		v = -1
	}
	return v
}

func newSoundBytes(sampleRate int, bitDepthInBytes int) []byte {
	b := make([]byte, sampleRate*4) // 1 second
	for i := 0; i < len(b)/(bitDepthInBytes*2); i++ {
		v := soundAt(float64(i) / float64(sampleRate))
		switch bitDepthInBytes {
		case 2:
			v16 := int16(v * (1<<15 - 1))
			b[4*i] = byte(v16)
			b[4*i+1] = byte(v16 >> 8)
			b[4*i+2] = byte(v16)
			b[4*i+3] = byte(v16 >> 8)
		case 4:
			v32 := math.Float32bits(float32(v))
			b[8*i] = byte(v32)
			b[8*i+1] = byte(v32 >> 8)
			b[8*i+2] = byte(v32 >> 16)
			b[8*i+3] = byte(v32 >> 24)
			b[8*i+4] = byte(v32)
			b[8*i+5] = byte(v32 >> 8)
			b[8*i+6] = byte(v32 >> 16)
			b[8*i+7] = byte(v32 >> 24)
		}
	}
	return b
}

func newSoundBytesForFrames(sampleRate int, frameCount int, bitDepthInBytes int) []byte {
	b := make([]byte, frameCount*bitDepthInBytes*2)
	for i := range frameCount {
		v := soundAt(float64(i) / float64(sampleRate))
		switch bitDepthInBytes {
		case 2:
			v16 := int16(v * (1<<15 - 1))
			b[4*i] = byte(v16)
			b[4*i+1] = byte(v16 >> 8)
			b[4*i+2] = byte(v16)
			b[4*i+3] = byte(v16 >> 8)
		case 4:
			v32 := math.Float32bits(float32(v))
			b[8*i] = byte(v32)
			b[8*i+1] = byte(v32 >> 8)
			b[8*i+2] = byte(v32 >> 16)
			b[8*i+3] = byte(v32 >> 24)
			b[8*i+4] = byte(v32)
			b[8*i+5] = byte(v32 >> 8)
			b[8*i+6] = byte(v32 >> 16)
			b[8*i+7] = byte(v32 >> 24)
		}
	}
	return b
}

type reader struct {
	r io.Reader
}

func (r *reader) Read(buf []byte) (int, error) {
	return r.r.Read(buf)
}

func TestResampling(t *testing.T) {
	cases := []struct {
		In  int
		Out int
	}{
		{
			In:  44100,
			Out: 48000,
		},
		{
			In:  48000,
			Out: 44100,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d to %d", c.In, c.Out), func(t *testing.T) {
			for _, bitDepthInBytes := range []int{2, 4} {
				t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
					for _, seek := range []bool{false, true} {
						t.Run(fmt.Sprintf("seek=%v", seek), func(t *testing.T) {
							inB := newSoundBytes(c.In, bitDepthInBytes)
							l := int64(len(inB))
							if !seek {
								l = 0
							}
							var src io.Reader = bytes.NewReader(inB)
							if !seek {
								src = &reader{r: src}
							}
							outS := convert.NewResampling(src, l, c.In, c.Out, bitDepthInBytes)
							var gotB []byte
							for {
								var buf [97]byte
								n, err := outS.Read(buf[:])
								gotB = append(gotB, buf[:n]...)
								if err != nil {
									if err != io.EOF {
										t.Fatal(err)
									}
									break
								}
								if seek {
									cur, err := outS.Seek(0, io.SeekCurrent)
									if err != nil {
										t.Fatal(err)
									}
									// Shifting by incomplete bytes should not affect the result.
									for i := 0; i < bitDepthInBytes*2; i++ {
										pos, err := outS.Seek(int64(i), io.SeekCurrent)
										if err != nil {
											t.Fatal(err)
										}
										if cur != pos {
											t.Errorf("cur: %d, pos: %d", cur, pos)
										}
									}
								}
							}
							wantB := newSoundBytes(c.Out, bitDepthInBytes)
							// 256 is an arbitrary number.
							// In most cases, len(gotB) must >= len(wantB), but there are some numerical errors.
							if len(gotB) < len(wantB)-256 {
								t.Errorf("len(gotB) >= len(wantB) - 256, but len(gotB) == %d, len(wantB) == %d", len(gotB), len(wantB))
							}
							for i := 0; i < len(gotB)/bitDepthInBytes; i++ {
								var got, want float64
								switch bitDepthInBytes {
								case 2:
									got = float64(int16(gotB[2*i])|(int16(gotB[2*i+1])<<8)) / (1<<15 - 1)
									if i < len(wantB)/2 {
										want = float64(int16(wantB[2*i])|(int16(wantB[2*i+1])<<8)) / (1<<15 - 1)
									}
								case 4:
									got = float64(math.Float32frombits(uint32(gotB[4*i]) | (uint32(gotB[4*i+1]) << 8) | (uint32(gotB[4*i+2]) << 16) | (uint32(gotB[4*i+3]) << 24)))
									if i < len(wantB)/4 {
										want = float64(math.Float32frombits(uint32(wantB[4*i]) | (uint32(wantB[4*i+1]) << 8) | (uint32(wantB[4*i+2]) << 16) | (uint32(wantB[4*i+3]) << 24)))
									}
								}
								if math.Abs(got-want) > 0.025 {
									t.Errorf("sample rate: %d, index: %d: got: %f, want: %f", c.Out, i, got, want)
								}
							}
						})
					}
				})
			}
		})
	}
}

func TestResamplingSeekNegative(t *testing.T) {
	for _, bitDepthInBytes := range []int{2, 4} {
		t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
			inB := newSoundBytes(44100, bitDepthInBytes)
			r := convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), 44100, 48000, bitDepthInBytes)

			if _, err := r.Seek(-1, io.SeekStart); err == nil {
				t.Error("Seek(-1, io.SeekStart): expected an error but got none")
			}
			if _, err := r.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Seek(-1, io.SeekCurrent); err == nil {
				t.Error("Seek(-1, io.SeekCurrent): expected an error but got none")
			}

			// io.SeekEnd must be relative to the end, not to the current position.
			if _, err := r.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			fromStart, err := r.Seek(-1000, io.SeekEnd)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.Seek(4000, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			fromMiddle, err := r.Seek(-1000, io.SeekEnd)
			if err != nil {
				t.Fatal(err)
			}
			if fromStart != fromMiddle {
				t.Errorf("Seek(-1000, io.SeekEnd) must not depend on the current position: from 0: %d, from 4000: %d", fromStart, fromMiddle)
			}

			// A negative result from io.SeekEnd must be rejected even from a non-zero position.
			if _, err := r.Seek(4000, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Seek(-r.Length()-1, io.SeekEnd); err == nil {
				t.Error("Seek(-Length()-1, io.SeekEnd): expected an error but got none")
			}

			// The stream must not be broken by the failed seeks.
			pos, err := r.Seek(0, io.SeekStart)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, int64(0); got != want {
				t.Errorf("Seek(0, io.SeekStart): got: %d, want: %d", got, want)
			}
			n, err := r.Read(make([]byte, 64))
			if err != nil {
				t.Fatalf("Read after Seek(0): %v", err)
			}
			if got, want := n, 64; got != want {
				t.Errorf("Read: got: %d, want: %d", got, want)
			}
		})
	}
}

// Issue #3352
func TestResamplingLen(t *testing.T) {
	buf := make([]byte, 8*48000)
	src := bytes.NewReader(buf)
	resampled := convert.NewResampling(src, int64(len(buf)), 48000, 96000, 4)
	if got, want := resampled.Length(), int64(len(buf)*2); got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}
	decodedBuf, err := io.ReadAll(resampled)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(decodedBuf), int(len(buf)*2); got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}
}

func newTestSoundBytes16(n int) []byte {
	b := make([]byte, n*4)
	for i := range n {
		b[4*i] = byte(i)
		b[4*i+1] = byte(i >> 8)
		b[4*i+2] = byte(255 - i)
		b[4*i+3] = byte((255 - i) >> 8)
	}
	return b
}

func TestResamplingSeekEndAbsolute(t *testing.T) {
	in := newTestSoundBytes16(1000)

	r := convert.NewResampling(bytes.NewReader(in), int64(len(in)), 44100, 44100, 2)
	got, err := r.Seek(-4, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if want := r.Length() - 4; got != want {
		t.Errorf("Seek(-4, io.SeekEnd) from the start: got %d, want %d", got, want)
	}

	r2 := convert.NewResampling(bytes.NewReader(in), int64(len(in)), 44100, 44100, 2)
	buf := make([]byte, 2000)
	for len(buf) > 0 {
		n, err := r2.Read(buf)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if n == 0 {
			t.Fatal("Read made no progress")
		}
		buf = buf[n:]
	}
	got2, err := r2.Seek(-4, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if got2 != got {
		t.Errorf("Seek(-4, io.SeekEnd) after reading 2000 bytes: got %d, want %d (the result must not depend on the current position)", got2, got)
	}

	b := make([]byte, 8)
	n, err := r2.Read(b)
	if n != 4 || (err != nil && err != io.EOF) {
		t.Errorf("Read at (Length-4) after Seek(-4, io.SeekEnd): got n=%d, err=%v; want n=4", n, err)
	}
}

func TestResamplingSeekEndZero(t *testing.T) {
	in := newTestSoundBytes16(500)
	r := convert.NewResampling(bytes.NewReader(in), int64(len(in)), 44100, 44100, 2)

	buf := make([]byte, 1000)
	if _, err := r.Read(buf); err != nil && err != io.EOF {
		t.Fatal(err)
	}

	got, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if want := r.Length(); got != want {
		t.Errorf("Seek(0, io.SeekEnd): got %d, want %d", got, want)
	}
	if want := int64(len(in)); r.Length() != want {
		t.Errorf("Length: got %d, want %d", r.Length(), want)
	}
}

func TestResamplingSeekUnknownLength(t *testing.T) {
	const (
		from            = 44100
		to              = 48000
		bitDepthInBytes = 2
		bytesPerSample  = bitDepthInBytes * 2
	)

	inB := newSoundBytes(from, bitDepthInBytes)

	// Read the entire stream whose length is known, in order to get the expected values.
	full, err := io.ReadAll(convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), from, to, bitDepthInBytes))
	if err != nil {
		t.Fatal(err)
	}

	// 0 as a size indicates that the length is unknown.
	r := convert.NewResampling(bytes.NewReader(inB), 0, from, to, bitDepthInBytes)

	const offset = 4000
	pos, err := r.Seek(offset, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pos, int64(offset); got != want {
		t.Errorf("Seek(%d, io.SeekStart): got %d, want %d", offset, got, want)
	}

	// An offset that is not aligned with a sample should be rounded down.
	pos, err = r.Seek(offset+bytesPerSample-1, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pos, int64(offset); got != want {
		t.Errorf("Seek(%d, io.SeekStart): got %d, want %d", offset+bytesPerSample-1, got, want)
	}

	pos, err = r.Seek(bytesPerSample, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pos, int64(offset+bytesPerSample); got != want {
		t.Errorf("Seek(%d, io.SeekCurrent): got %d, want %d", bytesPerSample, got, want)
	}

	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 400)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatal(err)
	}
	if got, want := buf, full[offset:offset+len(buf)]; !bytes.Equal(got, want) {
		t.Errorf("reading after Seek(%d, io.SeekStart) returned unexpected bytes", offset)
	}

	// Seeking from the end is not possible when the length is unknown.
	if _, err := r.Seek(0, io.SeekEnd); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Seek(0, io.SeekEnd): got %v, want an error matching errors.ErrUnsupported", err)
	}
}

func TestResamplingSeekInvalidWhence(t *testing.T) {
	const (
		from            = 44100
		to              = 48000
		bitDepthInBytes = 2
	)

	inB := newSoundBytes(from, bitDepthInBytes)
	r := convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), from, to, bitDepthInBytes)

	for _, whence := range []int{-1, 3, 100} {
		if _, err := r.Seek(0, whence); err == nil {
			t.Errorf("Seek(0, %d): got no error, want an error", whence)
		}
	}
}

func TestResamplingShortBuffer(t *testing.T) {
	cases := []struct {
		bitDepthInBytes int
		lens            []int
	}{
		{
			bitDepthInBytes: 2,
			lens:            []int{1, 2, 3},
		},
		{
			bitDepthInBytes: 4,
			lens:            []int{1, 2, 3, 4, 5, 6, 7},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("bitDepthInBytes=%d", c.bitDepthInBytes), func(t *testing.T) {
			src := make([]byte, 1024)
			r := convert.NewResampling(bytes.NewReader(src), int64(len(src)), 44100, 48000, c.bitDepthInBytes)

			if n, err := r.Read(nil); n != 0 || err != nil {
				t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
			}
			for _, l := range c.lens {
				if n, err := r.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
					t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
				}
			}
		})
	}
}

func readAllInChunks(t *testing.T, r io.Reader, chunkSizeInBytes int) []byte {
	t.Helper()

	var got []byte
	buf := make([]byte, chunkSizeInBytes)
	for {
		n, err := r.Read(buf)
		got = append(got, buf[:n]...)
		if err == io.EOF {
			return got
		}
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			t.Fatal("Read made no progress")
		}
	}
}

// Issue #3589
func TestResamplingReadBufferSize(t *testing.T) {
	cases := []struct {
		In  int
		Out int
	}{
		{
			In:  44100,
			Out: 44100,
		},
		{
			In:  44100,
			Out: 48000,
		},
		{
			In:  48000,
			Out: 44100,
		},
		{
			In:  48000,
			Out: 4000,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d to %d", c.In, c.Out), func(t *testing.T) {
			for _, bitDepthInBytes := range []int{2, 4} {
				t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
					bytesPerSample := bitDepthInBytes * 2
					// A buffer of this size makes the number of frames leave every possible remainder.
					const shortBufferInFrames = 8
					for frames := 200; frames < 200+shortBufferInFrames; frames++ {
						t.Run(fmt.Sprintf("frames=%d", frames), func(t *testing.T) {
							inB := newSoundBytes(c.In, bitDepthInBytes)[:frames*bytesPerSample]
							newResampling := func() *convert.Resampling {
								return convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), c.In, c.Out, bitDepthInBytes)
							}

							// A buffer that is long enough to hold the entire stream.
							r := newResampling()
							want := r.Length()
							gotLong := readAllInChunks(t, r, int(want)+bytesPerSample)
							if int64(len(gotLong)) != want {
								t.Errorf("reading with one long buffer: got %d bytes, want %d bytes", len(gotLong), want)
							}

							// A short buffer must deliver the same bytes.
							gotShort := readAllInChunks(t, newResampling(), shortBufferInFrames*bytesPerSample)
							if int64(len(gotShort)) != want {
								t.Errorf("reading with buffers of %d frames: got %d bytes, want %d bytes", shortBufferInFrames, len(gotShort), want)
							}
							if !bytes.Equal(gotShort, gotLong) {
								t.Errorf("reading with buffers of %d frames returned different bytes than reading with one long buffer", shortBufferInFrames)
							}
						})
					}
				})
			}
		})
	}
}

// Issue #3542
func TestResamplingUnknownLength(t *testing.T) {
	cases := []struct {
		In  int
		Out int
	}{
		{
			In:  44100,
			Out: 44100,
		},
		{
			In:  44100,
			Out: 48000,
		},
		{
			In:  48000,
			Out: 44100,
		},
		{
			In:  48000,
			Out: 4000,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d to %d", c.In, c.Out), func(t *testing.T) {
			for _, bitDepthInBytes := range []int{2, 4} {
				t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
					// The source is read in blocks of this size.
					blockSize := 4096 * bitDepthInBytes * 2
					// A size that is shorter than one sample, that is around a block boundary,
					// or that is a multiple of a block is the interesting case.
					for _, size := range []int{1, 100, blockSize - 2, blockSize, blockSize + 2, 40000} {
						t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
							inB := newSoundBytes(c.In, bitDepthInBytes)[:size]

							// A stream whose length is known gives the expected values.
							known := convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), c.In, c.Out, bitDepthInBytes)
							wantB, err := io.ReadAll(known)
							if err != nil {
								t.Fatal(err)
							}

							for _, seekable := range []bool{true, false} {
								t.Run(fmt.Sprintf("seekable=%v", seekable), func(t *testing.T) {
									newResampling := func() *convert.Resampling {
										var src io.Reader = bytes.NewReader(inB)
										if !seekable {
											src = &reader{r: src}
										}
										// 0 as a size indicates that the length is unknown.
										return convert.NewResampling(src, 0, c.In, c.Out, bitDepthInBytes)
									}

									r := newResampling()
									if got, want := r.Length(), int64(0); got != want {
										t.Errorf("Length: got %d, want %d", got, want)
									}
									gotB, err := io.ReadAll(r)
									if err != nil {
										t.Fatal(err)
									}
									if !bytes.Equal(gotB, wantB) {
										t.Errorf("io.ReadAll returned %d bytes, want the same %d bytes as a stream whose length is known", len(gotB), len(wantB))
									}

									// The read buffer size must not affect the result.
									gotShortB := readAllInChunks(t, newResampling(), 97)
									if !bytes.Equal(gotShortB, wantB) {
										t.Errorf("reading with buffers of 97 bytes returned %d bytes, want the same %d bytes as a stream whose length is known", len(gotShortB), len(wantB))
									}
								})
							}
						})
					}
				})
			}
		})
	}
}

func TestResamplingUnknownLengthEmptySource(t *testing.T) {
	for _, bitDepthInBytes := range []int{2, 4} {
		t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
			for _, seekable := range []bool{true, false} {
				t.Run(fmt.Sprintf("seekable=%v", seekable), func(t *testing.T) {
					var src io.Reader = bytes.NewReader(nil)
					if !seekable {
						src = &reader{r: src}
					}
					r := convert.NewResampling(src, 0, 44100, 48000, bitDepthInBytes)
					if n, err := r.Read(make([]byte, 4096)); n != 0 || err != io.EOF {
						t.Errorf("Read: got (%d, %v), want (0, %v)", n, err, io.EOF)
					}
				})
			}
		})
	}
}

var errStalled = errors.New("stalled")

// stalledReadSeeker reports its source's end as (0, nil) instead of io.EOF,
// and returns errStalled when zero reads repeat.
type stalledReadSeeker struct {
	src       io.ReadSeeker
	zeroReads int
}

func (s *stalledReadSeeker) Read(buf []byte) (int, error) {
	n, err := s.src.Read(buf)
	if err == io.EOF {
		s.zeroReads++
		if s.zeroReads > 16 {
			return 0, errStalled
		}
		return 0, nil
	}
	if n > 0 {
		s.zeroReads = 0
	}
	return n, err
}

func (s *stalledReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return s.src.Seek(offset, whence)
}

func TestResamplingZeroReadSource(t *testing.T) {
	const (
		from = 44100
		to   = 48000
	)

	for _, bitDepthInBytes := range []int{2, 4} {
		t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
			inB := newSoundBytes(from, bitDepthInBytes)

			wantB, err := io.ReadAll(convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), from, to, bitDepthInBytes))
			if err != nil {
				t.Fatal(err)
			}

			src := &stalledReadSeeker{src: bytes.NewReader(inB)}
			gotB, err := io.ReadAll(convert.NewResampling(src, int64(len(inB)), from, to, bitDepthInBytes))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotB, wantB) {
				t.Errorf("io.ReadAll returned %d bytes, want the same %d bytes as a source reporting io.EOF", len(gotB), len(wantB))
			}
		})
	}
}

func TestResamplingSeekAfterCachedBlock(t *testing.T) {
	const (
		from = 44100
		to   = 48000
	)

	for _, bitDepthInBytes := range []int{2, 4} {
		t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
			const blockFrames = 4096
			blockBytes := blockFrames * bitDepthInBytes * 2
			inB := newSoundBytesForFrames(from, 10*blockFrames, bitDepthInBytes)
			newResampling := func() *convert.Resampling {
				return convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), from, to, bitDepthInBytes)
			}
			outPos := func(pos int64) int64 {
				return pos * to / from
			}

			wantReader := newResampling()
			if _, err := wantReader.Seek(outPos(int64(2*blockBytes+blockBytes/2))+4096, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			want := make([]byte, 3*blockBytes)
			if _, err := io.ReadFull(wantReader, want); err != nil {
				t.Fatal(err)
			}

			gotReader := newResampling()
			read := func(pos int64, length int) []byte {
				t.Helper()
				if pos >= 0 {
					if _, err := gotReader.Seek(outPos(pos), io.SeekStart); err != nil {
						t.Fatal(err)
					}
				}
				buf := make([]byte, length)
				if _, err := io.ReadFull(gotReader, buf); err != nil {
					t.Fatal(err)
				}
				return buf
			}
			read(int64(2*blockBytes+blockBytes/2), 4096)
			read(int64(9*blockBytes+blockBytes/2), 4096)
			read(int64(2*blockBytes+blockBytes/2), 4096)
			got := read(-1, 3*blockBytes)

			if !bytes.Equal(got, want) {
				t.Error("resampled data differs from a fresh reader after seeking through a cached block")
			}
		})
	}
}

type midStreamStallingReadSeeker struct {
	src       io.ReadSeeker
	readCount int
}

func (s *midStreamStallingReadSeeker) Read(buf []byte) (int, error) {
	switch s.readCount {
	case 0:
		s.readCount++
		buf = buf[:len(buf)/2]
		return s.src.Read(buf)
	case 1:
		s.readCount++
		return 0, nil
	default:
		return s.src.Read(buf)
	}
}

func (s *midStreamStallingReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return s.src.Seek(offset, whence)
}

func TestResamplingZeroReadInMiddle(t *testing.T) {
	const (
		from = 44100
		to   = 48000
	)

	for _, bitDepthInBytes := range []int{2, 4} {
		t.Run(fmt.Sprintf("bitDepthInBytes=%d", bitDepthInBytes), func(t *testing.T) {
			const blockFrames = 4096
			blockBytes := blockFrames * bitDepthInBytes * 2
			inB := newSoundBytesForFrames(from, 5*blockFrames, bitDepthInBytes)

			want, err := io.ReadAll(convert.NewResampling(bytes.NewReader(inB), int64(len(inB)), from, to, bitDepthInBytes))
			if err != nil {
				t.Fatal(err)
			}

			src := &midStreamStallingReadSeeker{src: bytes.NewReader(inB)}
			got, err := io.ReadAll(convert.NewResampling(src, int64(len(inB)), from, to, bitDepthInBytes))
			if err != nil {
				t.Fatal(err)
			}

			start := int64(blockBytes)*to/from + 128
			if !bytes.Equal(got[start:], want[start:]) {
				t.Error("resampled data remains misaligned after a stalled source block")
			}
		})
	}
}
