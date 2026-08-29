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

var partialFrameTestCases = []struct {
	name string
	src  []byte
	mono bool
	fmt  convert.Format
	want []byte
}{
	{
		name: "mono S16 with a trailing byte",
		src:  []byte{0x80, 0x01, 0x02, 0x03, 0x04},
		mono: true,
		fmt:  convert.FormatS16,
		want: []byte{0x80, 0x01, 0x80, 0x01, 0x02, 0x03, 0x02, 0x03},
	},
	{
		name: "mono S16 aligned",
		src:  []byte{0x80, 0x01, 0x02, 0x03},
		mono: true,
		fmt:  convert.FormatS16,
		want: []byte{0x80, 0x01, 0x80, 0x01, 0x02, 0x03, 0x02, 0x03},
	},
	{
		name: "mono S16 four samples",
		src:  []byte{0x11, 0x11, 0x22, 0x22, 0x33, 0x33, 0x44, 0x44},
		mono: true,
		fmt:  convert.FormatS16,
		want: []byte{0x11, 0x11, 0x11, 0x11, 0x22, 0x22, 0x22, 0x22, 0x33, 0x33, 0x33, 0x33, 0x44, 0x44, 0x44, 0x44},
	},
	{
		name: "stereo S16 with a trailing byte",
		src:  []byte{0x80, 0x01, 0x02, 0x03, 0x04},
		mono: false,
		fmt:  convert.FormatS16,
		want: []byte{0x80, 0x01, 0x02, 0x03},
	},
	{
		name: "mono U8 aligned",
		src:  []byte{0x80, 0x81, 0x7f},
		mono: true,
		fmt:  convert.FormatU8,
		want: []byte{0x80, 0x00, 0x80, 0x00, 0x81, 0x01, 0x81, 0x01, 0x7f, 0xff, 0x7f, 0xff},
	},
	{
		name: "stereo U8 with a trailing byte",
		src:  []byte{0x80, 0x81, 0x7f, 0x7e, 0x01},
		mono: false,
		fmt:  convert.FormatU8,
		want: []byte{0x80, 0x00, 0x81, 0x01, 0x7f, 0xff, 0x7e, 0xfe},
	},
	{
		name: "mono S24 aligned",
		src:  []byte{1, 2, 3, 4, 5, 6},
		mono: true,
		fmt:  convert.FormatS24,
		want: []byte{2, 3, 2, 3, 5, 6, 5, 6},
	},
	{
		name: "stereo S24 aligned",
		src:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		mono: false,
		fmt:  convert.FormatS24,
		want: []byte{2, 3, 5, 6, 8, 9, 11, 12},
	},
	{
		name: "stereo S24 with a trailing partial frame",
		src:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
		mono: false,
		fmt:  convert.FormatS24,
		want: []byte{2, 3, 5, 6},
	},
}

func TestStereoI16ReadSeekerPartialFrame(t *testing.T) {
	for _, tc := range partialFrameTestCases {
		t.Run(tc.name, func(t *testing.T) {
			s := convert.NewStereoI16ReadSeeker(bytes.NewReader(tc.src), tc.mono, tc.fmt)
			buf := make([]byte, 64)
			n, err := s.Read(buf)
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			if !bytes.Equal(buf[:n], tc.want) {
				t.Errorf("Read: got %d bytes %x, want %x", n, buf[:n], tc.want)
			}
		})
	}
}

type shortReader struct {
	src   io.Reader
	chunk int
}

func (r *shortReader) Read(p []byte) (int, error) {
	if len(p) > r.chunk {
		p = p[:r.chunk]
	}
	return r.src.Read(p)
}

func (r *shortReader) Seek(offset int64, whence int) (int64, error) {
	return r.src.(io.Seeker).Seek(offset, whence)
}

func TestStereoI16ReadSeekerShortReads(t *testing.T) {
	for _, chunk := range []int{1, 2, 3, 5} {
		for _, tc := range partialFrameTestCases {
			t.Run(fmt.Sprintf("%s/chunk=%d", tc.name, chunk), func(t *testing.T) {
				s := convert.NewStereoI16ReadSeeker(&shortReader{src: bytes.NewReader(tc.src), chunk: chunk}, tc.mono, tc.fmt)
				var got []byte
				for {
					buf := make([]byte, 8)
					n, err := s.Read(buf)
					if n == 0 && err == nil {
						t.Fatalf("Read returned (0, nil) after %d bytes", len(got))
					}
					got = append(got, buf[:n]...)
					pos, seekErr := s.Seek(0, io.SeekCurrent)
					if seekErr != nil {
						t.Fatal(seekErr)
					}
					if pos != int64(len(got)) {
						t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", pos, len(got))
					}
					if err != nil {
						if err != io.EOF {
							t.Fatal(err)
						}
						break
					}
				}
				if !bytes.Equal(got, tc.want) {
					t.Errorf("got %x, want %x", got, tc.want)
				}

				if _, err := s.Seek(0, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				got = nil
				for {
					buf := make([]byte, 5)
					n, err := s.Read(buf)
					if n == 0 && err == nil {
						t.Fatalf("Read returned (0, nil) after %d bytes", len(got))
					}
					got = append(got, buf[:n]...)
					if err != nil {
						if err != io.EOF {
							t.Fatal(err)
						}
						break
					}
				}
				if !bytes.Equal(got, tc.want) {
					t.Errorf("after seeking: got %x, want %x", got, tc.want)
				}
			})
		}
	}
}

func TestStereoI16ReadSeekerTooShortBuffer(t *testing.T) {
	for _, tc := range []struct {
		bufSize int
		want    error
	}{
		{0, nil},
		{1, io.ErrShortBuffer},
		{3, io.ErrShortBuffer},
	} {
		t.Run(fmt.Sprintf("buf=%d", tc.bufSize), func(t *testing.T) {
			s := convert.NewStereoI16ReadSeeker(&shortReader{
				src:   bytes.NewReader([]byte{1, 2, 3, 4, 5, 6}),
				chunk: 5,
			}, true, convert.FormatS24)
			want := []byte{2, 3, 2, 3, 5, 6, 5, 6}

			buf := make([]byte, 8)
			n, err := s.Read(buf)
			if err != nil {
				t.Fatal(err)
			}
			got := buf[:n]

			n, err = s.Read(make([]byte, tc.bufSize))
			if n != 0 {
				t.Errorf("Read with a %d-byte buffer: got %d bytes, want 0", tc.bufSize, n)
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("Read with a %d-byte buffer: got error %v, want %v", tc.bufSize, err, tc.want)
			}

			for {
				buf := make([]byte, 8)
				n, err := s.Read(buf)
				got = append(got, buf[:n]...)
				if err != nil {
					if err != io.EOF {
						t.Fatal(err)
					}
					break
				}
			}
			if !bytes.Equal(got, want) {
				t.Errorf("got %x, want %x", got, want)
			}
		})
	}
}

func TestStereoI16SeekEndPartialFrame(t *testing.T) {
	for _, tc := range partialFrameTestCases {
		t.Run(tc.name, func(t *testing.T) {
			s := convert.NewStereoI16ReadSeeker(bytes.NewReader(tc.src), tc.mono, tc.fmt)
			pos, err := s.Seek(0, io.SeekEnd)
			if err != nil {
				t.Fatal(err)
			}
			want := len(tc.want)
			if int(pos) != want {
				t.Errorf("Seek(0, io.SeekEnd): got %d, want %d", pos, want)
			}
		})
	}
}

type failedSeeker struct {
	r   io.Reader
	err error
}

func (f *failedSeeker) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f *failedSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, f.err
}

func TestStereoI16ReadSeekerFailedSeekKeepsRemainder(t *testing.T) {
	src := []byte{1, 2, 3, 4, 5, 6}
	want := []byte{2, 3, 2, 3, 5, 6, 5, 6}
	s := convert.NewStereoI16ReadSeeker(&failedSeeker{
		r:   &shortReader{src: bytes.NewReader(src), chunk: 5},
		err: errors.New("seek failed"),
	}, true, convert.FormatS24)

	var got []byte
	for {
		buf := make([]byte, 8)
		n, err := s.Read(buf)
		got = append(got, buf[:n]...)
		if _, seekErr := s.Seek(0, io.SeekStart); seekErr == nil {
			t.Fatal("Seek: got nil error, want an error")
		}
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %x, want %x", got, want)
	}
}
