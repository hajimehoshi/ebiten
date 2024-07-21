// Copyright 2024 The Ebitengine Authors
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
	"fmt"
	"io"
	"math"
	"math/rand" // TODO: Use math/rand/v2 when the minimum supported version becomes Go 1.22.
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

func randFloat32s(n int) []float32 {
	r := make([]float32, n)
	for i := range r {
		r[i] = rand.Float32()*2 - 1
	}
	return r
}

func TestStereoF32(t *testing.T) {
	testCases := []struct {
		Name string
		In   []float32
	}{
		{
			Name: "nil",
			In:   nil,
		},
		{
			Name: "-1, 0, 1, 0",
			In:   []float32{-1, 0, 1, 0},
		},
		{
			Name: "8 0s",
			In:   []float32{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			Name: "random 256 values",
			In:   randFloat32s(256),
		},
		{
			Name: "random 65536 values",
			In:   randFloat32s(65536),
		},
	}
	for _, tc := range testCases {
		tc := tc
		for _, mono := range []bool{false, true} {
			mono := mono
			t.Run(fmt.Sprintf("%s (mono=%t)", tc.Name, mono), func(t *testing.T) {
				var inBytes, outBytes []byte
				for _, v := range tc.In {
					b := math.Float32bits(v)
					inBytes = append(inBytes, byte(b), byte(b>>8), byte(b>>16), byte(b>>24))
					if mono {
						// As the source is mono, the output should be stereo.
						outBytes = append(outBytes, byte(b), byte(b>>8), byte(b>>16), byte(b>>24), byte(b), byte(b>>8), byte(b>>16), byte(b>>24))
					} else {
						outBytes = append(outBytes, byte(b), byte(b>>8), byte(b>>16), byte(b>>24))
					}
				}
				s := convert.NewStereoF32(bytes.NewReader(inBytes), mono)
				var got []byte
				for {
					var buf [97]byte
					n, err := s.Read(buf[:])
					got = append(got, buf[:n]...)
					if err != nil {
						if err != io.EOF {
							t.Fatal(err)
						}
						break
					}
					if _, err := s.Seek(0, io.SeekCurrent); err != nil {
						if err != io.EOF {
							t.Fatal(err)
						}
						break
					}
				}
				want := outBytes
				if !bytes.Equal(got, want) {
					t.Errorf("got: %v, want: %v", got, want)
				}
			})
		}
	}
}
