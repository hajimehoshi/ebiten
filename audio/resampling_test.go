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
	"fmt"
	"io"
	"math"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func TestResampleReaderComposedSeek(t *testing.T) {
	source := func(t *testing.T, path string, seekable bool) io.Reader {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		r := bytes.NewReader(data)
		if seekable {
			return r
		}
		return struct{ io.Reader }{r}
	}
	for _, tc := range []struct {
		name      string
		f32       bool
		newSource func(*testing.T, bool) (io.Reader, int)
	}{
		{name: "WAV", newSource: func(t *testing.T, seekable bool) (io.Reader, int) {
			s, err := wav.DecodeWithoutResampling(source(t, "../examples/resources/audio/jab.wav", seekable))
			if err != nil {
				t.Fatal(err)
			}
			return s, s.SampleRate()
		}},
		{name: "Vorbis", newSource: func(t *testing.T, seekable bool) (io.Reader, int) {
			s, err := vorbis.DecodeWithoutResampling(source(t, "vorbis/test_mono.ogg", seekable))
			if err != nil {
				t.Fatal(err)
			}
			return s, s.SampleRate()
		}},
		{name: "MP3", newSource: func(t *testing.T, seekable bool) (io.Reader, int) {
			s, err := mp3.DecodeWithoutResampling(source(t, "../examples/resources/audio/ragtime.mp3", seekable))
			if err != nil {
				t.Fatal(err)
			}
			return s, s.SampleRate()
		}},
		{name: "nested", newSource: func(t *testing.T, seekable bool) (io.Reader, int) {
			data := make([]byte, 8000)
			for i := range len(data) / 2 {
				binary.LittleEndian.PutUint16(data[2*i:], uint16(i*31))
			}
			var src io.Reader = bytes.NewReader(data)
			if !seekable {
				src = struct{ io.Reader }{src}
			}
			return audio.ResampleReader(src, int64(len(data)), 44100, 48000), 48000
		}},
		{name: "WAV F32", f32: true, newSource: func(t *testing.T, seekable bool) (io.Reader, int) {
			s, err := wav.DecodeF32(source(t, "../examples/resources/audio/jab.wav", seekable))
			if err != nil {
				t.Fatal(err)
			}
			return s, s.SampleRate()
		}},
		{name: "nested F32", f32: true, newSource: func(t *testing.T, seekable bool) (io.Reader, int) {
			data := make([]byte, 8000)
			for i := range len(data) / 4 {
				binary.LittleEndian.PutUint32(data[4*i:], math.Float32bits(float32(i%100)/100))
			}
			var src io.Reader = bytes.NewReader(data)
			if !seekable {
				src = struct{ io.Reader }{src}
			}
			return audio.ResampleReaderF32(src, int64(len(data)), 44100, 48000), 48000
		}},
	} {
		for _, seekable := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/seekable=%t", tc.name, seekable), func(t *testing.T) {
				newReader := func() io.ReadSeeker {
					src, rate := tc.newSource(t, seekable)
					if tc.f32 {
						return audio.ResampleReaderF32(src, 0, rate, 32000).(io.ReadSeeker)
					}
					return audio.ResampleReader(src, 0, rate, 32000).(io.ReadSeeker)
				}
				want := make([]byte, 64)
				if _, err := io.ReadFull(newReader(), want); err != nil {
					t.Fatal(err)
				}
				r := newReader()
				got := make([]byte, 64)
				if _, err := io.ReadFull(r, got[:16]); err != nil {
					t.Fatal(err)
				}
				if seekable {
					if pos, err := r.Seek(0, io.SeekStart); pos != 0 || err != nil {
						t.Fatalf("Seek: got (%d, %v), want (0, nil)", pos, err)
					}
					if _, err := io.ReadFull(r, got); err != nil {
						t.Fatal(err)
					}
				} else {
					// Reject both a cached rewind and a distant seek immediately, without changing the output.
					for _, seek := range []struct {
						offset int64
						whence int
					}{{0, io.SeekStart}, {1 << 20, io.SeekStart}, {0, io.SeekCurrent}, {0, io.SeekEnd}} {
						if _, err := r.Seek(seek.offset, seek.whence); !errors.Is(err, errors.ErrUnsupported) {
							t.Errorf("Seek(%d, %d): got %v, want errors.ErrUnsupported", seek.offset, seek.whence, err)
						}
					}
					if _, err := io.ReadFull(r, got[16:]); err != nil {
						t.Fatal(err)
					}
				}
				if !bytes.Equal(got, want) {
					t.Error("output changed after seeking")
				}
			})
		}
	}
}
