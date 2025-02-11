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
	"io"
	"testing"

	"github.com/jfreymuth/oggvorbis"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

var (
	//go:embed test_mono.ogg
	test_mono_ogg []byte

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
	// On the other hand, the original vorbis package is monoral.
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
	// On the other hand, the original vorbis package is monoral.
	// As Length() represents the number of samples,
	// this needs to be doubled by 4 (= bytes in 32bits).
	if got, want := s.Length(), r.Length()*2*4; got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
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
