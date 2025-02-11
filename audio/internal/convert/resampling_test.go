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
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

func soundAt(timeInSecond float64) float64 {
	const freq = 220

	amp := []float64{1.0, 0.8, 0.6, 0.4, 0.2}
	v := 0.0
	for j := 0; j < len(amp); j++ {
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
		c := c
		t.Run(fmt.Sprintf("%d to %d", c.In, c.Out), func(t *testing.T) {
			for _, bitDepthInBytes := range []int{2, 4} {
				bitDepthInBytes := bitDepthInBytes
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
