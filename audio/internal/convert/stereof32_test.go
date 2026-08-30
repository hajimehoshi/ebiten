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
	"math/rand/v2"
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
		t.Run(tc.Name, func(t *testing.T) {
			for _, mono := range []bool{false, true} {
				t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
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
						// Shifting by incomplete bytes should not affect the result.
						for i := range 4 * 2 {
							if _, err := s.Seek(int64(i), io.SeekCurrent); err != nil {
								if err != io.EOF {
									t.Fatal(err)
								}
								break
							}
						}
					}
					want := outBytes
					if !bytes.Equal(got, want) {
						t.Errorf("got: %v, want: %v", got, want)
					}
				})
			}
		})
	}
}

func TestStereoF32SeekEnd(t *testing.T) {
	for _, mono := range []bool{false, true} {
		t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
			const frames = 100
			// A monaural source has half the bytes per frame.
			bytesPerFrame := 8
			if mono {
				bytesPerFrame /= 2
			}
			src := bytes.NewReader(make([]byte, frames*bytesPerFrame))
			s := convert.NewStereoF32(src, mono)

			pos, err := s.Seek(0, io.SeekEnd)
			if err != nil {
				t.Fatal(err)
			}
			// The stream is always stereo f32 (8 bytes per frame).
			want := frames * 8
			if pos != int64(want) {
				t.Errorf("Seek(0, io.SeekEnd): got %d, want %d", pos, want)
			}
		})
	}
}

func TestStereoF32SeekEndUnalignedSource(t *testing.T) {
	for _, mono := range []bool{false, true} {
		t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
			// A monaural source has half the bytes per frame.
			bytesPerFrame := 8
			if mono {
				bytesPerFrame /= 2
			}
			// Append an incomplete frame to the source.
			for extra := 1; extra < bytesPerFrame; extra++ {
				t.Run(fmt.Sprintf("extra=%d", extra), func(t *testing.T) {
					const frames = 100
					in := randBytes(frames*bytesPerFrame + extra)
					s := convert.NewStereoF32(bytes.NewReader(in), mono)

					// The stream is always stereo f32 (8 bytes per frame).
					whole := make([]byte, frames*8)
					if _, err := io.ReadFull(s, whole); err != nil {
						t.Fatal(err)
					}

					for _, framesFromEnd := range []int64{0, 1, 10, frames} {
						offset := -8 * framesFromEnd
						pos, err := s.Seek(offset, io.SeekEnd)
						if err != nil {
							t.Fatal(err)
						}
						if want := int64(frames*8) + offset; pos != want {
							t.Errorf("Seek(%d, io.SeekEnd): got %d, want %d", offset, pos, want)
						}
						got := make([]byte, framesFromEnd*8)
						if _, err := io.ReadFull(s, got); err != nil {
							t.Fatal(err)
						}
						if want := whole[int64(len(whole))-framesFromEnd*8:]; !bytes.Equal(got, want) {
							t.Errorf("reading after Seek(%d, io.SeekEnd): got % x, want % x", offset, got, want)
						}
					}
				})
			}
		})
	}
}
