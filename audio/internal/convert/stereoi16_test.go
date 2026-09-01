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
	"math/rand/v2"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

func TestStereoI16FromSigned16Bits(t *testing.T) {
	testCases := []struct {
		Name string
		In   []int16
	}{
		{
			Name: "nil",
			In:   nil,
		},
		{
			Name: "-1, 0, 1, 0",
			In:   []int16{-1, 0, 1, 0},
		},
		{
			Name: "8 0s",
			In:   []int16{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			Name: "random 256 values",
			In:   randInt16s(256),
		},
		{
			Name: "random 65536 values",
			In:   randInt16s(65536),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			for _, mono := range []bool{false, true} {
				t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
					var inBytes, outBytes []byte
					for _, v := range tc.In {
						inBytes = append(inBytes, byte(v), byte(v>>8))
						if mono {
							// As the source is mono, the output should be stereo.
							outBytes = append(outBytes, byte(v), byte(v>>8), byte(v), byte(v>>8))
						} else {
							outBytes = append(outBytes, byte(v), byte(v>>8))
						}
					}
					s := convert.NewStereoI16ReadSeeker(bytes.NewReader(inBytes), mono, convert.FormatS16)
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
						for i := range 2 * 2 {
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

func randBytes(n int) []byte {
	r := make([]byte, n)
	for i := range r {
		r[i] = byte(rand.IntN(256))
	}
	return r
}

func TestStereoI16FromUnsigned8Bits(t *testing.T) {
	testCases := []struct {
		Name string
		In   []byte
	}{
		{
			Name: "nil",
			In:   nil,
		},
		{
			Name: "1, 0, 1, 0",
			In:   []byte{1, 0, 1, 0},
		},
		{
			Name: "8 0s",
			In:   []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			Name: "random 256 values",
			In:   randBytes(256),
		},
		{
			Name: "random 65536 values",
			In:   randBytes(65536),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			for _, mono := range []bool{false, true} {
				t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
					inBytes := tc.In
					var outBytes []byte
					for _, v := range tc.In {
						v16 := int16(int(v)*0x101 - (1 << 15))
						if mono {
							// As the source is mono, the output should be stereo.
							outBytes = append(outBytes, byte(v16), byte(v16>>8), byte(v16), byte(v16>>8))
						} else {
							outBytes = append(outBytes, byte(v16), byte(v16>>8))
						}
					}
					s := convert.NewStereoI16ReadSeeker(bytes.NewReader(inBytes), mono, convert.FormatU8)
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
						for i := range 2 * 2 {
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

func randInts(n int) []int {
	r := make([]int, n)
	for i := range r {
		r[i] = rand.Int()
	}
	return r
}

func TestStereoI16FromSigned24Bits(t *testing.T) {
	testCases := []struct {
		Name string
		In   []int
	}{
		{
			Name: "nil",
			In:   nil,
		},
		{
			Name: "-1, 0, 1, 0",
			In:   []int{-1, 0, 1, 0},
		},
		{
			Name: "8 0s",
			In:   []int{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			Name: "random 256 values",
			In:   randInts(256),
		},
		{
			Name: "random 65536 values",
			In:   randInts(65536),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			for _, mono := range []bool{false, true} {
				t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
					var inBytes, outBytes []byte
					for _, v := range tc.In {
						inBytes = append(inBytes, byte(v), byte(v>>8), byte(v>>16))
						if mono {
							// As the source is mono, the output should be stereo.
							outBytes = append(outBytes, byte(v>>8), byte(v>>16), byte(v>>8), byte(v>>16))
						} else {
							outBytes = append(outBytes, byte(v>>8), byte(v>>16))
						}
					}
					s := convert.NewStereoI16ReadSeeker(bytes.NewReader(inBytes), mono, convert.FormatS24)
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
						for i := range 2 * 2 {
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

func TestStereoI16SeekEnd(t *testing.T) {
	testCases := []struct {
		name          string
		format        convert.Format
		bytesPerFrame int
	}{
		{
			name:          "S16",
			format:        convert.FormatS16,
			bytesPerFrame: 4,
		},
		{
			name:          "U8",
			format:        convert.FormatU8,
			bytesPerFrame: 2,
		},
		{
			name:          "S24",
			format:        convert.FormatS24,
			bytesPerFrame: 6,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, mono := range []bool{false, true} {
				t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
					const frames = 100
					// A monaural source has half the bytes per frame.
					bytesPerFrame := tc.bytesPerFrame
					if mono {
						bytesPerFrame /= 2
					}
					src := bytes.NewReader(make([]byte, frames*bytesPerFrame))
					s := convert.NewStereoI16ReadSeeker(src, mono, tc.format)

					pos, err := s.Seek(0, io.SeekEnd)
					if err != nil {
						t.Fatal(err)
					}
					// The stream is always stereo i16 (4 bytes per frame).
					want := frames * 4
					if pos != int64(want) {
						t.Errorf("Seek(0, io.SeekEnd): got %d, want %d", pos, want)
					}
				})
			}
		})
	}
}

func TestStereoI16SeekEndUnalignedSource(t *testing.T) {
	testCases := []struct {
		name               string
		format             convert.Format
		sourceBytesPerUnit int
	}{
		{
			name:               "S16",
			format:             convert.FormatS16,
			sourceBytesPerUnit: 2,
		},
		{
			name:               "U8",
			format:             convert.FormatU8,
			sourceBytesPerUnit: 1,
		},
		{
			name:               "S24",
			format:             convert.FormatS24,
			sourceBytesPerUnit: 3,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, mono := range []bool{false, true} {
				t.Run(fmt.Sprintf("mono=%t", mono), func(t *testing.T) {
					bytesPerFrame := tc.sourceBytesPerUnit
					if !mono {
						bytesPerFrame *= 2
					}
					// Append an incomplete frame to the source.
					for extra := 1; extra < bytesPerFrame; extra++ {
						t.Run(fmt.Sprintf("extra=%d", extra), func(t *testing.T) {
							const frames = 100
							in := randBytes(frames*bytesPerFrame + extra)
							s := convert.NewStereoI16ReadSeeker(bytes.NewReader(in), mono, tc.format)

							// The stream is always stereo i16 (4 bytes per frame).
							whole := make([]byte, frames*4)
							if _, err := io.ReadFull(s, whole); err != nil {
								t.Fatal(err)
							}

							for _, framesFromEnd := range []int64{0, 1, 10, frames} {
								offset := -4 * framesFromEnd
								pos, err := s.Seek(offset, io.SeekEnd)
								if err != nil {
									t.Fatal(err)
								}
								if want := int64(frames*4) + offset; pos != want {
									t.Errorf("Seek(%d, io.SeekEnd): got %d, want %d", offset, pos, want)
								}
								got := make([]byte, framesFromEnd*4)
								if _, err := io.ReadFull(s, got); err != nil {
									t.Fatal(err)
								}
								if want := whole[int64(len(whole))-framesFromEnd*4:]; !bytes.Equal(got, want) {
									t.Errorf("reading after Seek(%d, io.SeekEnd): got % x, want % x", offset, got, want)
								}
							}
						})
					}
				})
			}
		})
	}
}

func TestStereoI16ShortBuffer(t *testing.T) {
	for _, mono := range []bool{false, true} {
		for _, format := range []convert.Format{convert.FormatU8, convert.FormatS16, convert.FormatS24} {
			t.Run(fmt.Sprintf("mono=%t,format=%d", mono, format), func(t *testing.T) {
				r := convert.NewStereoI16ReadSeeker(bytes.NewReader(make([]byte, 64)), mono, format)

				if n, err := r.Read(nil); n != 0 || err != nil {
					t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
				}
				for _, l := range []int{1, 2, 3} {
					if n, err := r.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
						t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
					}
				}
			})
		}
	}
}

var stereoI16Formats = []struct {
	name   string
	format convert.Format
	// unit is the byte size of one sample of one channel.
	unit int
}{
	{
		name:   "U8",
		format: convert.FormatU8,
		unit:   1,
	},
	{
		name:   "S16",
		format: convert.FormatS16,
		unit:   2,
	},
	{
		name:   "S24",
		format: convert.FormatS24,
		unit:   3,
	},
}

// stereoI16Bytes converts src to stereo int16 bytes, ignoring an incomplete frame.
func stereoI16Bytes(src []byte, mono bool, format convert.Format, unit int) []byte {
	frameSize := unit
	if !mono {
		frameSize *= 2
	}
	var dst []byte
	for i := range len(src) / frameSize {
		frame := src[i*frameSize : (i+1)*frameSize]
		for ch := range 2 {
			sample := frame[:unit]
			if !mono && ch == 1 {
				sample = frame[unit:]
			}
			switch format {
			case convert.FormatU8:
				v := int16(int(sample[0])*0x101 - (1 << 15))
				dst = append(dst, byte(v), byte(v>>8))
			case convert.FormatS16:
				dst = append(dst, sample[0], sample[1])
			case convert.FormatS24:
				dst = append(dst, sample[1], sample[2])
			}
		}
	}
	return dst
}

func TestStereoI16ShortReads(t *testing.T) {
	for _, f := range stereoI16Formats {
		for _, mono := range []bool{false, true} {
			frameSize := f.unit
			if !mono {
				frameSize *= 2
			}
			// Vary the source length so that the source ends in the middle of a frame.
			for extra := range frameSize {
				for _, maxN := range []int{1, 2, 3, 5} {
					for _, bufLen := range []int{4, 13, 64} {
						t.Run(fmt.Sprintf("format=%s,mono=%t,extra=%d,maxN=%d,bufLen=%d", f.name, mono, extra, maxN, bufLen), func(t *testing.T) {
							src := randBytes(10*frameSize + extra)
							s := convert.NewStereoI16ReadSeeker(&dribbleReader{r: bytes.NewReader(src), maxN: maxN}, mono, f.format)

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
								if n%4 != 0 {
									t.Fatalf("Read: got %d bytes, want a multiple of 4", n)
								}
								got = append(got, buf[:n]...)
								if err == io.EOF {
									break
								}
								if err != nil {
									t.Fatal(err)
								}
							}

							if want := stereoI16Bytes(src, mono, f.format, f.unit); !bytes.Equal(got, want) {
								t.Errorf("got % x, want % x", got, want)
							}
						})
					}
				}
			}
		}
	}
}

func TestStereoI16SeekCurrentAfterShortRead(t *testing.T) {
	for _, f := range stereoI16Formats {
		for _, mono := range []bool{false, true} {
			t.Run(fmt.Sprintf("format=%s,mono=%t", f.name, mono), func(t *testing.T) {
				frameSize := f.unit
				if !mono {
					frameSize *= 2
				}
				src := randBytes(20*frameSize + 1)
				// maxN is not a multiple of the source frame size, so an incomplete frame
				// remains buffered after a Read.
				s := convert.NewStereoI16ReadSeeker(&dribbleReader{r: bytes.NewReader(src), maxN: 5}, mono, f.format)

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

				if want := stereoI16Bytes(src, mono, f.format, f.unit); !bytes.Equal(got, want) {
					t.Errorf("got % x, want % x", got, want)
				}
			})
		}
	}
}

func TestStereoI16SeekOutOfRangeLeavesStreamIntact(t *testing.T) {
	const frames = 100
	// The stream is always stereo i16 (4 bytes per frame).
	const streamLen = frames * 4
	// pos is a frame boundary inside the stream.
	const pos = 40

	testCases := []struct {
		name   string
		offset int64
		whence int
	}{
		{
			name:   "SeekStartNegative",
			offset: -4,
			whence: io.SeekStart,
		},
		{
			name:   "SeekStartPastEnd",
			offset: streamLen + 4,
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
			offset: -streamLen - 4,
			whence: io.SeekEnd,
		},
		{
			name:   "SeekEndPastEnd",
			offset: 4,
			whence: io.SeekEnd,
		},
	}

	for _, f := range stereoI16Formats {
		for _, mono := range []bool{false, true} {
			frameSize := f.unit
			if !mono {
				frameSize *= 2
			}
			src := randBytes(frames * frameSize)
			whole := stereoI16Bytes(src, mono, f.format, f.unit)
			for _, tc := range testCases {
				t.Run(fmt.Sprintf("format=%s,mono=%t,%s", f.name, mono, tc.name), func(t *testing.T) {
					// maxN is not a multiple of the source frame size, so an incomplete frame
					// remains buffered after a Read.
					s := convert.NewStereoI16ReadSeeker(&boundedSeeker{r: &dribbleReader{r: bytes.NewReader(src), maxN: 5}, size: int64(len(src))}, mono, f.format)

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
}

func TestStereoI16SeekSmallNegativePosition(t *testing.T) {
	const frames = 100

	for _, f := range stereoI16Formats {
		for _, mono := range []bool{false, true} {
			t.Run(fmt.Sprintf("format=%s,mono=%t", f.name, mono), func(t *testing.T) {
				frameSize := f.unit
				if !mono {
					frameSize *= 2
				}
				src := randBytes(frames * frameSize)
				s := convert.NewStereoI16ReadSeeker(&boundedSeeker{r: bytes.NewReader(src), size: int64(len(src))}, mono, f.format)
				// The stream is always stereo i16 (4 bytes per frame).
				const streamLen = frames * 4

				// An offset in (-4, 0) is rounded toward the frame boundary, so the requested
				// position must be resolved and checked before the rounding, whichever whence
				// it comes from.
				for _, offset := range []int64{-1, -2, -3, -4} {
					if _, err := s.Seek(offset, io.SeekStart); err == nil {
						t.Errorf("Seek(%d, io.SeekStart): got no error, want an error", offset)
					}
					if _, err := s.Seek(offset, io.SeekCurrent); err == nil {
						t.Errorf("Seek(%d, io.SeekCurrent) at 0: got no error, want an error", offset)
					}
					if _, err := s.Seek(-streamLen+offset, io.SeekEnd); err == nil {
						t.Errorf("Seek(-%d%+d, io.SeekEnd): got no error, want an error", streamLen, offset)
					}
				}

				// The stream must not be broken by the rejected seeks.
				if pos, err := s.Seek(0, io.SeekCurrent); err != nil {
					t.Fatal(err)
				} else if pos != 0 {
					t.Errorf("Seek(0, io.SeekCurrent): got: %d, want: 0", pos)
				}
			})
		}
	}
}
