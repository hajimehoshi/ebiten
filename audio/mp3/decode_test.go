// Copyright 2026 The Ebitengine Authors
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

package mp3_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	resources "github.com/hajimehoshi/ebiten/v2/examples/resources/audio"
)

var mp3Decoders = []struct {
	name           string
	f              func(io.Reader) (*mp3.Stream, error)
	bytesPerSample int64
}{
	{
		name: "DecodeWithoutResampling",
		f: func(r io.Reader) (*mp3.Stream, error) {
			return mp3.DecodeWithoutResampling(r)
		},
		bytesPerSample: 4,
	},
	{
		name: "DecodeF32",
		f: func(r io.Reader) (*mp3.Stream, error) {
			return mp3.DecodeF32(r)
		},
		bytesPerSample: 8,
	},
	{
		name: "DecodeWithSampleRate",
		f: func(r io.Reader) (*mp3.Stream, error) {
			// ragtime.mp3 is 48000Hz, so this resamples the stream.
			return mp3.DecodeWithSampleRate(44100, r)
		},
		bytesPerSample: 4,
	},
}

func TestSeekNegativePosition(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}

			// An offset in (-4, 0) is rounded toward the sample boundary on some of the paths,
			// so its sign must be checked before the rounding.
			for _, offset := range []int64{-1, -2, -3, -4, -100} {
				if _, err := s.Seek(offset, io.SeekStart); err == nil {
					t.Errorf("Seek(%d, io.SeekStart): got no error, want an error", offset)
				}
			}

			// A position resolved from the other whences must be rejected as well.
			if _, err := s.Seek(-1, io.SeekCurrent); err == nil {
				t.Error("Seek(-1, io.SeekCurrent): got no error, want an error")
			}
			if _, err := s.Seek(-s.Length()-1, io.SeekEnd); err == nil {
				t.Error("Seek(-Length()-1, io.SeekEnd): got no error, want an error")
			}

			// The stream must not be broken by the rejected seeks.
			pos, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, int64(0); got != want {
				t.Errorf("Seek(0, io.SeekCurrent): got: %d, want: %d", got, want)
			}
			buf := make([]byte, 64)
			if _, err := io.ReadFull(s, buf); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSeekRejectedPositionRecovery(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			const position = 1024
			want := make([]byte, position+64)
			if _, err := io.ReadFull(s, want); err != nil {
				t.Fatal(err)
			}

			for _, tc := range []struct {
				name   string
				offset int64
				whence int
			}{
				{
					name:   "CurrentOverflow",
					offset: math.MaxInt64,
					whence: io.SeekCurrent,
				},
				{
					name:   "EndOverflow",
					offset: math.MaxInt64,
					whence: io.SeekEnd,
				},
				{
					name:   "StartNegative",
					offset: -1,
					whence: io.SeekStart,
				},
				{
					name:   "CurrentNegative",
					offset: -position - 1,
					whence: io.SeekCurrent,
				},
				{
					name:   "EndNegative",
					offset: -s.Length() - 1,
					whence: io.SeekEnd,
				},
				{
					name:   "CurrentMinInt64",
					offset: math.MinInt64,
					whence: io.SeekCurrent,
				},
				{
					name:   "EndMinInt64",
					offset: math.MinInt64,
					whence: io.SeekEnd,
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
					if err != nil {
						t.Fatal(err)
					}
					if _, err := io.CopyN(io.Discard, s, position); err != nil {
						t.Fatal(err)
					}
					if _, err := s.Seek(tc.offset, tc.whence); err == nil {
						t.Errorf("Seek(%d, %d): got no error, want an error", tc.offset, tc.whence)
					}
					pos, err := s.Seek(0, io.SeekCurrent)
					if err != nil {
						t.Fatal(err)
					}
					if pos != position {
						t.Errorf("position after rejected seek: got %d, want %d", pos, position)
					}
					got := make([]byte, len(want)-position)
					if _, err := io.ReadFull(s, got); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(got, want[position:]) {
						t.Error("data after rejected seek differs from uninterrupted decoding")
					}
				})
			}
		})
	}
}

func TestSeekInvalidWhence(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			for _, whence := range []int{-1, 3, 100} {
				if _, err := s.Seek(0, whence); err == nil {
					t.Errorf("Seek(0, %d): got no error, want an error", whence)
				}
			}
		})
	}
}

// readerOnly hides the Seek method of the wrapped reader.
type readerOnly struct {
	io.Reader
}

func TestSeekNonSeekableSource(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(readerOnly{bytes.NewReader(resources.Ragtime_mp3)})
			if err != nil {
				t.Fatal(err)
			}
			if got, want := s.Length(), int64(-1); got != want {
				t.Errorf("Length(): got: %d, want: %d", got, want)
			}
			for _, whence := range []int{io.SeekStart, io.SeekCurrent, io.SeekEnd} {
				if _, err := s.Seek(0, whence); !errors.Is(err, errors.ErrUnsupported) {
					t.Errorf("Seek(0, %d): got %v, want an error matching errors.ErrUnsupported", whence, err)
				}
			}

			// The stream must still be readable after the rejected seeks.
			buf := make([]byte, 64)
			if _, err := io.ReadFull(s, buf); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSeekSeekableSource(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			if s.Length() <= 0 {
				t.Fatalf("Length(): got: %d, want a positive value", s.Length())
			}

			// Read the head of the stream, seek back to the start, and read it again.
			// The two reads must be the same.
			want := make([]byte, 1024)
			if _, err := io.ReadFull(s, want); err != nil {
				t.Fatal(err)
			}
			pos, err := s.Seek(0, io.SeekStart)
			if err != nil {
				t.Fatal(err)
			}
			if pos != 0 {
				t.Errorf("Seek(0, io.SeekStart): got: %d, want: 0", pos)
			}
			got := make([]byte, len(want))
			if _, err := io.ReadFull(s, got); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Error("the data read after Seek(0, io.SeekStart) differs from the first read")
			}
			pos, err = s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, int64(len(want)); got != want {
				t.Errorf("Seek(0, io.SeekCurrent): got: %d, want: %d", got, want)
			}
		})
	}
}

func TestDecodeWithSampleRateNonSeekableSource(t *testing.T) {
	const ragtimeMP3FrameSizeInBytes = 384
	for _, frames := range []int{1, 10, 32} {
		t.Run(fmt.Sprintf("frames=%d", frames), func(t *testing.T) {
			src := resources.Ragtime_mp3[:frames*ragtimeMP3FrameSizeInBytes]

			seekable, err := mp3.DecodeWithSampleRate(44100, bytes.NewReader(src))
			if err != nil {
				t.Fatal(err)
			}
			want, err := io.ReadAll(seekable)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := int64(len(want)), seekable.Length(); got != want {
				t.Errorf("io.ReadAll from a seekable source returned %d bytes, want Length() %d", got, want)
			}

			nonSeekable, err := mp3.DecodeWithSampleRate(44100, readerOnly{bytes.NewReader(src)})
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(nonSeekable)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("io.ReadAll from a non-seekable source returned %d bytes, want the same %d bytes as from a seekable source", len(got), len(want))
			}
		})
	}
}

func TestDecodeWithSampleRateInvalidSampleRate(t *testing.T) {
	for _, sampleRate := range []int{0, -1} {
		if _, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(resources.Ragtime_mp3)); err == nil {
			t.Errorf("mp3.DecodeWithSampleRate(%d): got no error, want an error", sampleRate)
		}
	}
}

// Issue #3619
func TestSeekPastEnd(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			if s.Length() <= 0 {
				t.Fatalf("Length(): got %d, want a positive value", s.Length())
			}

			for _, offset := range []int64{s.Length(), s.Length() + 1000, 10 * s.Length()} {
				pos, err := s.Seek(offset, io.SeekStart)
				if err != nil {
					t.Errorf("Seek(%d, io.SeekStart): %v", offset, err)
					continue
				}
				if got, want := pos, offset; got != want {
					t.Errorf("Seek(%d, io.SeekStart): got %d, want %d", offset, got, want)
				}
				if n, err := s.Read(make([]byte, 8192)); n != 0 || !errors.Is(err, io.EOF) {
					t.Errorf("Read after Seek(%d, io.SeekStart): got (%d, %v), want (0, %v)", offset, n, err, io.EOF)
				}
				pos, err = s.Seek(0, io.SeekCurrent)
				if err != nil {
					t.Errorf("Seek(0, io.SeekCurrent) after Seek(%d, io.SeekStart): %v", offset, err)
					continue
				}
				if got, want := pos, offset; got != want {
					t.Errorf("Seek(0, io.SeekCurrent) after Seek(%d, io.SeekStart): got %d, want %d", offset, got, want)
				}
			}

			pos, err := s.Seek(0, io.SeekEnd)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, s.Length(); got != want {
				t.Errorf("Seek(0, io.SeekEnd): got %d, want %d", got, want)
			}
			if n, err := s.Read(make([]byte, 8192)); n != 0 || !errors.Is(err, io.EOF) {
				t.Errorf("Read after Seek(0, io.SeekEnd): got (%d, %v), want (0, %v)", n, err, io.EOF)
			}

			pos, err = s.Seek(-8, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, s.Length()-8; got != want {
				t.Errorf("Seek(-8, io.SeekCurrent) at the end: got %d, want %d", got, want)
			}
			if n, err := s.Read(make([]byte, 8192)); n != 8 || (err != nil && !errors.Is(err, io.EOF)) {
				t.Errorf("Read after Seek(-8, io.SeekCurrent) at the end: got (%d, %v), want (8, nil or %v)", n, err, io.EOF)
			}

			pos, err = s.Seek(0, io.SeekStart)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, int64(0); got != want {
				t.Errorf("Seek(0, io.SeekStart): got %d, want %d", got, want)
			}
			if n, err := s.Read(make([]byte, 8192)); n == 0 || err != nil {
				t.Errorf("Read after Seek(0, io.SeekStart): got (%d, %v), want data and no error", n, err)
			}
		})
	}
}

func TestSeekRounding(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			size := decode.bytesPerSample
			base := s.Length() / 2 / size * size
			want := make([]byte, 4*size)
			if _, err := s.Seek(base, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadFull(s, want); err != nil {
				t.Fatal(err)
			}

			for r := int64(1); r < size; r++ {
				for _, tc := range []struct {
					name   string
					from   int64
					offset int64
					whence int
				}{
					{
						name:   "SeekStart",
						from:   0,
						offset: base + r,
						whence: io.SeekStart,
					},
					{
						name:   "SeekCurrent",
						from:   0,
						offset: base + r,
						whence: io.SeekCurrent,
					},
					{
						name:   "SeekCurrentBackward",
						from:   base + size,
						offset: r - size,
						whence: io.SeekCurrent,
					},
					{
						name:   "SeekEnd",
						from:   0,
						offset: base + r - s.Length(),
						whence: io.SeekEnd,
					},
				} {
					if _, err := s.Seek(tc.from, io.SeekStart); err != nil {
						t.Fatal(err)
					}
					pos, err := s.Seek(tc.offset, tc.whence)
					if err != nil {
						t.Fatal(err)
					}
					if got, want := pos, base; got != want {
						t.Errorf("%s to %d: got %d, want %d", tc.name, base+r, got, want)
					}
					got := make([]byte, len(want))
					if _, err := io.ReadFull(s, got); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(got, want) {
						t.Errorf("%s to %d: the data read differs from the data at %d", tc.name, base+r, base)
					}
				}

				pos, err := s.Seek(s.Length()+r, io.SeekStart)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := pos, s.Length(); got != want {
					t.Errorf("Seek(Length()+%d, io.SeekStart): got %d, want %d", r, got, want)
				}
			}
		})
	}
}

func TestSeekCurrentAfterUnalignedRead(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			const skip = 1 << 15
			ref, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			want := make([]byte, skip+8192)
			if _, err := io.ReadFull(ref, want); err != nil {
				t.Fatal(err)
			}

			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadFull(s, make([]byte, skip)); err != nil {
				t.Fatal(err)
			}
			n, err := s.Read(make([]byte, decode.bytesPerSample+1))
			if err != nil {
				t.Fatal(err)
			}
			pos, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, int64(skip+n); got != want {
				t.Errorf("Seek(0, io.SeekCurrent) after reading %d bytes: got %d, want %d", skip+n, got, want)
			}
			got := make([]byte, 4096)
			if _, err := io.ReadFull(s, got); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want[skip+n:skip+n+len(got)]) {
				t.Errorf("reading after Seek(0, io.SeekCurrent) at %d: the data differs from reading without the query", skip+n)
			}
		})
	}
}

// seekCountingReader is an io.ReadSeeker that counts its seeks.
type seekCountingReader struct {
	r     io.ReadSeeker
	seeks int
}

func (s *seekCountingReader) Read(buf []byte) (int, error) {
	return s.r.Read(buf)
}

func (s *seekCountingReader) Seek(offset int64, whence int) (int64, error) {
	s.seeks++
	return s.r.Seek(offset, whence)
}

func TestSeekCurrentAfterShortBufferRead(t *testing.T) {
	for _, decode := range mp3Decoders {
		t.Run(decode.name, func(t *testing.T) {
			const skip = 1 << 15
			ref, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			want := make([]byte, skip+8192)
			if _, err := io.ReadFull(ref, want); err != nil {
				t.Fatal(err)
			}

			r := &seekCountingReader{
				r: bytes.NewReader(resources.Ragtime_mp3),
			}
			s, err := decode.f(r)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadFull(s, make([]byte, skip)); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Read(make([]byte, 1)); !errors.Is(err, io.ErrShortBuffer) {
				t.Fatalf("Read(a buffer of 1 byte): got %v, want %v", err, io.ErrShortBuffer)
			}

			seeks := r.seeks
			pos, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, int64(skip); got != want {
				t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", got, want)
			}
			if r.seeks != seeks {
				t.Errorf("Seek(0, io.SeekCurrent) sought the source %d times, want 0", r.seeks-seeks)
			}
			got := make([]byte, len(want)-skip)
			if _, err := io.ReadFull(s, got); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want[skip:]) {
				t.Error("reading after Seek(0, io.SeekCurrent): the data differs from reading without the query")
			}

			tail := 16 * decode.bytesPerSample
			if _, err := s.Seek(s.Length()-tail, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadFull(s, make([]byte, tail)); err != nil {
				t.Fatal(err)
			}
			seeks = r.seeks
			pos, err = s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pos, s.Length(); got != want {
				t.Errorf("Seek(0, io.SeekCurrent) at the end: got %d, want %d", got, want)
			}
			if n, err := s.Read(make([]byte, 64)); n != 0 || !errors.Is(err, io.EOF) {
				t.Errorf("Read after Seek(0, io.SeekCurrent) at the end: got (%d, %v), want (0, %v)", n, err, io.EOF)
			}
			if r.seeks != seeks {
				t.Errorf("Seek(0, io.SeekCurrent) at the end sought the source %d times, want 0", r.seeks-seeks)
			}
		})
	}
}

func readWithSizes(t *testing.T, r io.Reader, sizes []int, bytesPerSample int, limit int) []byte {
	t.Helper()
	var got []byte
	for i := 0; len(got) < limit; i++ {
		size := sizes[i%len(sizes)]
		buf := make([]byte, size)
		n, err := r.Read(buf)
		if n%bytesPerSample != 0 {
			t.Errorf("Read(a buffer of %d bytes) at %d: got %d bytes, want a multiple of %d", size, len(got), n, bytesPerSample)
		}
		got = append(got, buf[:n]...)
		if errors.Is(err, io.EOF) {
			break
		}
		if size < bytesPerSample && errors.Is(err, io.ErrShortBuffer) {
			continue
		}
		if err != nil {
			t.Fatalf("Read(a buffer of %d bytes) at %d: %v", size, len(got), err)
		}
		if n == 0 {
			t.Fatalf("Read(a buffer of %d bytes) at %d: got (0, <nil>)", size, len(got))
		}
	}
	return got
}

func TestReadWholeSamples(t *testing.T) {
	decoders := append(mp3Decoders, struct {
		name           string
		f              func(io.Reader) (*mp3.Stream, error)
		bytesPerSample int64
	}{
		name: "SameSampleRate",
		f: func(r io.Reader) (*mp3.Stream, error) {
			return mp3.DecodeWithSampleRate(48000, r)
		},
		bytesPerSample: 4,
	})
	const limit = 1 << 16
	for _, decode := range decoders {
		t.Run(decode.name, func(t *testing.T) {
			ref, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			want := make([]byte, limit)
			if _, err := io.ReadFull(ref, want); err != nil {
				t.Fatal(err)
			}

			s, err := decode.f(bytes.NewReader(resources.Ragtime_mp3))
			if err != nil {
				t.Fatal(err)
			}
			got := readWithSizes(t, s, []int{1, 5, 3, 6, 2, 7, 9, 13, 4, 10, 8, 1027}, int(decode.bytesPerSample), limit)
			if len(got) < limit {
				t.Fatalf("got %d bytes, want at least %d", len(got), limit)
			}
			if !bytes.Equal(got[:limit], want) {
				t.Error("the data differs from reading with aligned buffers")
			}
		})
	}
}
