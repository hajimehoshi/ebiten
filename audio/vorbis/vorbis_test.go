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
	"testing"

	"github.com/jfreymuth/oggvorbis"

	"github.com/hajimehoshi/ebiten/v2/audio"
	. "github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

var audioContext = audio.NewContext(44100)

func TestMono(t *testing.T) {
	bs := test_mono_ogg

	s, err := DecodeWithSampleRate(audioContext.SampleRate(), bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	r, err := oggvorbis.NewReader(bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	// Stream decoded by audio/vorbis.DecodeWithSampleRate() is always 16bit stereo.
	got := s.Length()

	// On the other hand, the original vorbis package is monoral.
	// As Length() represents the number of samples,
	// this needs to be doubled by 2 (= bytes in 16bits).
	want := r.Length() * 2 * 2

	if got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}
}

func TestTooShort(t *testing.T) {
	bs := test_tooshort_ogg

	s, err := DecodeWithSampleRate(audioContext.SampleRate(), bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}

	got := s.Length()
	want := int64(79424)
	if got != want {
		t.Errorf("s.Length(): got: %d, want: %d", got, want)
	}
}
