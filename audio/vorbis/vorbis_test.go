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

package vorbis_test

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"testing"

	"github.com/jfreymuth/oggvorbis"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

var (
	// test_mono.ogg is in the public domain.
	// https://commons.wikimedia.org/wiki/File:Coins_dropped_on_wooden_floor.ogg
	//go:embed test_mono.ogg
	test_mono_ogg []byte

	// test_stereo.ogg is in the public domain.
	// https://commons.wikimedia.org/wiki/File:Example_sound_file_in_Ogg_Vorbis_format.ogg
	//go:embed test_stereo.ogg
	test_stereo_ogg []byte

	// test_tooshort.ogg is in the public domain.
	// https://opengameart.org/content/jumping-man-sounds
	//go:embed test_tooshort.ogg
	test_tooshort_ogg []byte
)

var audioContext = audio.NewContext(44100)

func TestMono(t *testing.T) {
	bs := test_mono_ogg

	s, err := vorbis.DecodeWithSampleRate(audioContext.SampleRate(), bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	r, err := oggvorbis.NewReader(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	// Stream decoded by audio/vorbis.DecodeWithSampleRate() is always 16bit stereo.
	// On the other hand, the original vorbis package is monaural.
	// As Length() represents the number of samples,
	// this needs to be doubled by 2 (= bytes in 16bits).
	if got, want := s.Length(), r.Length()*2*2; got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}

	if got, want := s.SampleRate(), audioContext.SampleRate(); got != want {
		t.Errorf("s.SampleRate(): got: %d, want: %d", got, want)
	}
}

func TestMonoF32(t *testing.T) {
	bs := test_mono_ogg

	s, err := vorbis.DecodeF32(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	r, err := oggvorbis.NewReader(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	// Stream decoded by audio/vorbis.DecodeF32() is always 32bit float stereo.
	// On the other hand, the original vorbis package is monaural.
	// As Length() represents the number of samples,
	// this needs to be doubled by 4 (= bytes in 32bits).
	if got, want := s.Length(), r.Length()*2*4; got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}
}

func TestStereoF32Seek(t *testing.T) {
	bs := test_stereo_ogg

	s, err := vorbis.DecodeF32(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	// A stream decoded by audio/vorbis.DecodeF32() is always 32bit float stereo.
	const sampleSize = 2 * 4

	pos, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pos, s.Length(); got != want {
		t.Errorf("s.Seek(0, io.SeekEnd): got: %d, want: %d", got, want)
	}

	off := s.Length() / 2 / sampleSize * sampleSize
	pos, err = s.Seek(off, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pos, off; got != want {
		t.Errorf("s.Seek(%d, io.SeekStart): got: %d, want: %d", off, got, want)
	}

	const delta = sampleSize * 16
	pos, err = s.Seek(delta, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pos, off+delta; got != want {
		t.Errorf("s.Seek(%d, io.SeekCurrent): got: %d, want: %d", int64(delta), got, want)
	}
}

func TestTooShort(t *testing.T) {
	bs := test_tooshort_ogg

	s, err := vorbis.DecodeWithSampleRate(audioContext.SampleRate(), bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := s.Length(), int64(79424); got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}

	if got, want := s.SampleRate(), audioContext.SampleRate(); got != want {
		t.Errorf("s.SampleRate(): got: %d, want: %d", got, want)
	}
}

func TestTooShortF32(t *testing.T) {
	bs := test_tooshort_ogg

	s, err := vorbis.DecodeF32(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := s.Length(), int64(158848); got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}
}

type reader struct {
	r io.Reader
}

func (r *reader) Read(buf []byte) (int, error) {
	return r.r.Read(buf)
}

func TestNonSeeker(t *testing.T) {
	bs := test_tooshort_ogg

	s, err := vorbis.DecodeWithSampleRate(audioContext.SampleRate(), &reader{r: bytes.NewReader(bs)})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := s.Length(), int64(0); got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}

	if got, want := s.SampleRate(), audioContext.SampleRate(); got != want {
		t.Errorf("s.SampleRate(): got: %d, want: %d", got, want)
	}

	buf, err := io.ReadAll(s)
	if err != nil {
		t.Errorf("io.ReadAll: %v", err)
	}
	if len(buf) == 0 {
		t.Errorf("len(buf): got: %d, want: > 0", len(buf))
	}
}

func TestNonSeekerF32(t *testing.T) {
	bs := test_tooshort_ogg

	s, err := vorbis.DecodeF32(&reader{r: bytes.NewReader(bs)})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := s.Length(), int64(0); got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}

	buf, err := io.ReadAll(s)
	if err != nil {
		t.Errorf("io.ReadAll: %v", err)
	}
	if len(buf) == 0 {
		t.Errorf("len(buf): got: %d, want: > 0", len(buf))
	}
}

func TestMonoI16SeekEnd(t *testing.T) {
	bs := test_mono_ogg

	s, err := vorbis.DecodeWithoutResampling(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if want := s.Length(); got != want {
		t.Errorf("Seek(0, io.SeekEnd): got %d, want %d (stream length)", got, want)
	}
}

func TestMonoF32Seek(t *testing.T) {
	bs := test_mono_ogg

	s, err := vorbis.DecodeF32(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("SeekEnd", func(t *testing.T) {
		got, err := s.Seek(0, io.SeekEnd)
		if err != nil {
			t.Fatal(err)
		}
		if want := s.Length(); got != want {
			t.Errorf("Seek(0, io.SeekEnd): got %d, want %d (stream length)", got, want)
		}
	})
	t.Run("SeekEndWithNegativeOffset", func(t *testing.T) {
		got, err := s.Seek(-8, io.SeekEnd)
		if err != nil {
			t.Fatal(err)
		}
		if want := s.Length() - 8; got != want {
			t.Errorf("Seek(-8, io.SeekEnd): got %d, want %d", got, want)
		}
	})
	t.Run("SeekStart", func(t *testing.T) {
		got, err := s.Seek(0, io.SeekStart)
		if err != nil {
			t.Fatal(err)
		}
		if want := int64(0); got != want {
			t.Errorf("Seek(0, io.SeekStart): got %d, want %d", got, want)
		}
		got, err = s.Seek(8, io.SeekStart)
		if err != nil {
			t.Fatal(err)
		}
		if want := int64(8); got != want {
			t.Errorf("Seek(8, io.SeekStart): got %d, want %d", got, want)
		}
	})
}

func TestSeekInvalidWhence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		decode func(src io.Reader) (*vorbis.Stream, error)
	}{
		{"I16", vorbis.DecodeWithoutResampling},
		{"F32", vorbis.DecodeF32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.decode(bytes.NewReader(test_mono_ogg))
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

func TestSeekNegativePosition(t *testing.T) {
	for _, file := range []struct {
		name string
		bs   []byte
	}{
		{
			name: "Mono",
			bs:   test_mono_ogg,
		},
		{
			name: "Stereo",
			bs:   test_stereo_ogg,
		},
	} {
		t.Run(file.name, func(t *testing.T) {
			for _, decode := range []struct {
				name string
				f    func(io.Reader) (*vorbis.Stream, error)
			}{
				{
					name: "I16",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						return vorbis.DecodeWithoutResampling(r)
					},
				},
				{
					name: "F32",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						return vorbis.DecodeF32(r)
					},
				},
				{
					name: "Resampling",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						// The test files are 44100Hz, so this resamples the stream.
						return vorbis.DecodeWithSampleRate(48000, r)
					},
				},
			} {
				t.Run(decode.name, func(t *testing.T) {
					s, err := decode.f(bytes.NewReader(file.bs))
					if err != nil {
						t.Fatal(err)
					}

					if _, err := s.Seek(-100, io.SeekStart); err == nil {
						t.Error("Seek(-100, io.SeekStart): expected an error but got none")
					}
					if _, err := s.Seek(0, io.SeekStart); err != nil {
						t.Fatal(err)
					}
					if _, err := s.Seek(-1000, io.SeekCurrent); err == nil {
						t.Error("Seek(-1000, io.SeekCurrent): expected an error but got none")
					}
					// A negative result from io.SeekEnd must be rejected even from a non-zero position.
					if _, err := s.Seek(4096, io.SeekStart); err != nil {
						t.Fatal(err)
					}
					if _, err := s.Seek(-s.Length()-1024, io.SeekEnd); err == nil {
						t.Error("Seek(-Length()-1024, io.SeekEnd): expected an error but got none")
					}

					// The stream must not be broken by the failed seeks.
					pos, err := s.Seek(0, io.SeekStart)
					if err != nil {
						t.Fatalf("Seek(0, io.SeekStart) after failed seeks: %v", err)
					}
					if got, want := pos, int64(0); got != want {
						t.Errorf("Seek(0, io.SeekStart): got: %d, want: %d", got, want)
					}
					buf := make([]byte, 64)
					n, err := s.Read(buf)
					if err != nil {
						t.Fatalf("Read after Seek(0): %v", err)
					}
					if got, want := n, len(buf); got != want {
						t.Errorf("Read: got: %d, want: %d", got, want)
					}
				})
			}
		})
	}
}

func TestStereoI16SeekUnalignedPosition(t *testing.T) {
	bs := test_stereo_ogg

	s, err := vorbis.DecodeWithoutResampling(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	// A seek result must be the actual frame-aligned position in the stream.
	got, err := s.Seek(5, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(4); got != want {
		t.Errorf("Seek(5, io.SeekStart): got: %d, want: %d", got, want)
	}

	buf := make([]byte, 8)
	n, err := s.Read(buf)
	if err != nil {
		t.Fatalf("Read after Seek(5, io.SeekStart): %v", err)
	}
	if got, want := n, len(buf); got != want {
		t.Errorf("Read: got: %d, want: %d", got, want)
	}
}

func TestStereoF32SeekUnalignedNegativeOffset(t *testing.T) {
	bs := test_stereo_ogg

	s, err := vorbis.DecodeF32(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	// A negative offset from io.SeekEnd must be aligned toward the start,
	// as well as the int16 stream does.
	got, err := s.Seek(-5, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	const sampleSize = 2 * 4
	if want := (s.Length() - 5) / sampleSize * sampleSize; got != want {
		t.Errorf("Seek(-5, io.SeekEnd): got: %d, want: %d", got, want)
	}
}

func TestDecodeF32ShortBuffer(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{
			name: "mono",
			data: test_mono_ogg,
		},
		{
			name: "stereo",
			data: test_stereo_ogg,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := vorbis.DecodeF32(bytes.NewReader(c.data))
			if err != nil {
				t.Fatal(err)
			}

			if n, err := s.Read(nil); n != 0 || err != nil {
				t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
			}
			for _, l := range []int{1, 2, 3, 4, 5, 6, 7} {
				if n, err := s.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
					t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
				}
			}
		})
	}
}

func TestDecodeI16ShortBuffer(t *testing.T) {
	for _, file := range []struct {
		name string
		bs   []byte
	}{
		{
			name: "Mono",
			bs:   test_mono_ogg,
		},
		{
			name: "Stereo",
			bs:   test_stereo_ogg,
		},
	} {
		t.Run(file.name, func(t *testing.T) {
			for _, decode := range []struct {
				name string
				f    func(io.Reader) (*vorbis.Stream, error)
			}{
				{
					name: "I16",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						return vorbis.DecodeWithoutResampling(r)
					},
				},
				{
					name: "Resampling",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						// The test files are 44100Hz, so this resamples the stream.
						return vorbis.DecodeWithSampleRate(48000, r)
					},
				},
			} {
				t.Run(decode.name, func(t *testing.T) {
					s, err := decode.f(bytes.NewReader(file.bs))
					if err != nil {
						t.Fatal(err)
					}

					if n, err := s.Read(nil); n != 0 || err != nil {
						t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
					}
					for _, l := range []int{1, 2, 3} {
						if n, err := s.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
							t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
						}
					}

					// The rejected reads must not consume the stream.
					n, err := io.Copy(io.Discard, s)
					if err != nil {
						t.Fatal(err)
					}
					if want := s.Length(); n != want {
						t.Errorf("the stream delivered %d bytes, want %d", n, want)
					}
				})
			}
		})
	}
}

func TestSeekSmallNegativePosition(t *testing.T) {
	for _, file := range []struct {
		name string
		bs   []byte
	}{
		{
			name: "Mono",
			bs:   test_mono_ogg,
		},
		{
			name: "Stereo",
			bs:   test_stereo_ogg,
		},
	} {
		t.Run(file.name, func(t *testing.T) {
			for _, decode := range []struct {
				name string
				f    func(io.Reader) (*vorbis.Stream, error)
			}{
				{
					name: "I16",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						return vorbis.DecodeWithoutResampling(r)
					},
				},
				{
					name: "F32",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						return vorbis.DecodeF32(r)
					},
				},
				{
					name: "Resampling",
					f: func(r io.Reader) (*vorbis.Stream, error) {
						// The test files are 44100Hz, so this resamples the stream.
						return vorbis.DecodeWithSampleRate(48000, r)
					},
				},
			} {
				t.Run(decode.name, func(t *testing.T) {
					s, err := decode.f(bytes.NewReader(file.bs))
					if err != nil {
						t.Fatal(err)
					}

					// An offset in (-8, 0) is rounded toward the frame boundary before it
					// reaches the decoder, so the requested position must be resolved and
					// checked before the rounding, whichever whence it comes from.
					for _, offset := range []int64{-1, -2, -3, -4, -5, -6, -7, -8} {
						if _, err := s.Seek(offset, io.SeekStart); err == nil {
							t.Errorf("Seek(%d, io.SeekStart): got no error, want an error", offset)
						}
						if _, err := s.Seek(offset, io.SeekCurrent); err == nil {
							t.Errorf("Seek(%d, io.SeekCurrent) at 0: got no error, want an error", offset)
						}
						if _, err := s.Seek(-s.Length()+offset, io.SeekEnd); err == nil {
							t.Errorf("Seek(-Length()%+d, io.SeekEnd): got no error, want an error", offset)
						}
					}

					// The stream must not be broken by the rejected seeks.
					buf := make([]byte, 64)
					if _, err := io.ReadFull(s, buf); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}
