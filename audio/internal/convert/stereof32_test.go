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
	"errors"
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

func TestStereoF32ShortBuffer(t *testing.T) {
	for _, mono := range []bool{false, true} {
		t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
			r := convert.NewStereoF32(bytes.NewReader(make([]byte, 64)), mono)

			if n, err := r.Read(nil); n != 0 || err != nil {
				t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
			}
			for _, l := range []int{1, 2, 3, 4, 5, 6, 7} {
				if n, err := r.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
					t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
				}
			}
		})
	}
}

// stereoF32Bytes converts src to stereo float32 bytes, ignoring an incomplete frame.
func stereoF32Bytes(src []byte, mono bool) []byte {
	frameSize := 8
	if mono {
		frameSize = 4
	}
	var dst []byte
	for i := range len(src) / frameSize {
		frame := src[i*frameSize : (i+1)*frameSize]
		dst = append(dst, frame...)
		if mono {
			dst = append(dst, frame...)
		}
	}
	return dst
}

func TestStereoF32ShortReads(t *testing.T) {
	for _, mono := range []bool{false, true} {
		frameSize := 8
		if mono {
			frameSize = 4
		}
		// Vary the source length so that the source ends in the middle of a frame.
		for extra := range frameSize {
			for _, maxN := range []int{1, 2, 3, 5} {
				for _, bufLen := range []int{8, 21, 64} {
					t.Run(fmt.Sprintf("mono=%t,extra=%d,maxN=%d,bufLen=%d", mono, extra, maxN, bufLen), func(t *testing.T) {
						src := randBytes(10*frameSize + extra)
						s := convert.NewStereoF32(&dribbleReader{r: bytes.NewReader(src), maxN: maxN}, mono)

						var got []byte
						for {
							buf := make([]byte, bufLen)
							for i := range buf {
								buf[i] = 0xff
							}
							n, err := s.Read(buf)
							if n == 0 && err == nil {
								t.Fatal("Read: got (0, <nil>), want a non-zero byte count or an error")
							}
							if n%8 != 0 {
								t.Fatalf("Read: got %d bytes, want a multiple of 8", n)
							}
							got = append(got, buf[:n]...)
							if err == io.EOF {
								break
							}
							if err != nil {
								t.Fatal(err)
							}
						}

						if want := stereoF32Bytes(src, mono); !bytes.Equal(got, want) {
							t.Errorf("got % x, want % x", got, want)
						}
					})
				}
			}
		}
	}
}

func TestStereoF32SeekCurrentAfterShortRead(t *testing.T) {
	for _, mono := range []bool{false, true} {
		t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
			frameSize := 8
			if mono {
				frameSize = 4
			}
			src := randBytes(20*frameSize + 1)
			// maxN is not a multiple of the source frame size, so an incomplete frame
			// remains buffered after a Read.
			s := convert.NewStereoF32(&dribbleReader{r: bytes.NewReader(src), maxN: 5}, mono)

			buf := make([]byte, 64)
			n, err := s.Read(buf)
			if err != nil {
				t.Fatal(err)
			}
			got := append([]byte(nil), buf[:n]...)

			pos, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if pos != int64(len(got)) {
				t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", pos, len(got))
			}

			for {
				n, err := s.Read(buf)
				got = append(got, buf[:n]...)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
			}

			if want := stereoF32Bytes(src, mono); !bytes.Equal(got, want) {
				t.Errorf("got % x, want % x", got, want)
			}
		})
	}
}

func TestStereoF32SeekOutOfRangeLeavesStreamIntact(t *testing.T) {
	const frames = 100
	// The stream is always stereo f32 (8 bytes per frame).
	const streamLen = frames * 8
	// pos is a frame boundary inside the stream.
	const pos = 80

	testCases := []struct {
		name   string
		offset int64
		whence int
	}{
		{
			name:   "SeekStartNegative",
			offset: -8,
			whence: io.SeekStart,
		},
		{
			name:   "SeekStartPastEnd",
			offset: streamLen + 8,
			whence: io.SeekStart,
		},
		{
			name:   "SeekCurrentNegative",
			offset: -streamLen,
			whence: io.SeekCurrent,
		},
		{
			name:   "SeekCurrentPastEnd",
			offset: streamLen,
			whence: io.SeekCurrent,
		},
		{
			name:   "SeekEndBeforeStart",
			offset: -streamLen - 8,
			whence: io.SeekEnd,
		},
		{
			name:   "SeekEndPastEnd",
			offset: 8,
			whence: io.SeekEnd,
		},
	}

	for _, mono := range []bool{false, true} {
		// A monaural source has half the bytes per frame.
		bytesPerFrame := 8
		if mono {
			bytesPerFrame /= 2
		}
		src := randBytes(frames * bytesPerFrame)
		whole := stereoF32Bytes(src, mono)
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("mono=%t,%s", mono, tc.name), func(t *testing.T) {
				// maxN is not a multiple of the source frame size, so an incomplete frame
				// remains buffered after a Read.
				s := convert.NewStereoF32(&boundedSeeker{r: &dribbleReader{r: bytes.NewReader(src), maxN: 5}, size: int64(len(src))}, mono)

				if _, err := s.Seek(pos, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				n, err := s.Read(make([]byte, 64))
				if err != nil {
					t.Fatal(err)
				}
				want := int64(pos + n)

				if _, err := s.Seek(tc.offset, tc.whence); err == nil {
					t.Errorf("Seek(%d, %d): got no error, want an error", tc.offset, tc.whence)
				}

				cur, err := s.Seek(0, io.SeekCurrent)
				if err != nil {
					t.Fatal(err)
				}
				if cur != want {
					t.Errorf("Seek(0, io.SeekCurrent) after a rejected Seek(%d, %d): got %d, want %d", tc.offset, tc.whence, cur, want)
				}

				got := make([]byte, 16)
				if _, err := io.ReadFull(s, got); err != nil {
					t.Fatal(err)
				}
				if w := whole[want : want+int64(len(got))]; !bytes.Equal(got, w) {
					t.Errorf("reading after a rejected Seek(%d, %d): got % x, want % x", tc.offset, tc.whence, got, w)
				}
			})
		}
	}
}
