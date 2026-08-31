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

package opus_test

import (
	"bytes"
	"io"
	"math/bits"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/opus"
	resources "github.com/hajimehoshi/ebiten/v2/examples/resources/audio"
)

// skipOn32Bit skips the test on 32-bit platforms, where decoding never returns.
//
// The decoder in github.com/kazzmir/opus-go is transpiled C accessing struct fields at
// offsets fixed for 64-bit layouts, and its range decoder loops forever on ILP32.
//
// TODO: Remove this skip after the decoder works on 32-bit platforms (#3621).
func skipOn32Bit(t *testing.T) {
	t.Helper()
	if bits.UintSize == 32 {
		t.Skip("decoding Opus hangs on 32-bit platforms (#3621)")
	}
}

var opusDecoders = []struct {
	name string
	f    func(io.Reader) (*opus.Stream, error)
}{
	{
		name: "Decode",
		f: func(r io.Reader) (*opus.Stream, error) {
			return opus.Decode(r)
		},
	},
	{
		name: "DecodeF32",
		f: func(r io.Reader) (*opus.Stream, error) {
			return opus.DecodeF32(r)
		},
	},
}

func TestSeekNegativePosition(t *testing.T) {
	skipOn32Bit(t)

	for _, decode := range opusDecoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_opus))
			if err != nil {
				t.Fatal(err)
			}

			// An offset in (-4, 0) is rounded toward the sample boundary by the decoder, so its
			// sign must be checked before the rounding.
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
	skipOn32Bit(t)

	for _, decode := range opusDecoders {
		t.Run(decode.name, func(t *testing.T) {
			s, err := decode.f(bytes.NewReader(resources.Ragtime_opus))
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
