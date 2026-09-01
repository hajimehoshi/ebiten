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
	"io"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	resources "github.com/hajimehoshi/ebiten/v2/examples/resources/audio"
)

var mp3Decoders = []struct {
	name string
	f    func(io.Reader) (*mp3.Stream, error)
}{
	{
		name: "DecodeWithoutResampling",
		f: func(r io.Reader) (*mp3.Stream, error) {
			return mp3.DecodeWithoutResampling(r)
		},
	},
	{
		name: "DecodeF32",
		f: func(r io.Reader) (*mp3.Stream, error) {
			return mp3.DecodeF32(r)
		},
	},
	{
		name: "DecodeWithSampleRate",
		f: func(r io.Reader) (*mp3.Stream, error) {
			// ragtime.mp3 is 48000Hz, so this resamples the stream.
			return mp3.DecodeWithSampleRate(44100, r)
		},
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
