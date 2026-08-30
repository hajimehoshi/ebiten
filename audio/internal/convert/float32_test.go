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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

func randInt16s(n int) []int16 {
	r := make([]int16, n)
	for i := range r {
		r[i] = int16(rand.IntN(1<<16) - (1 << 15))
	}
	return r
}

func TestFloat32(t *testing.T) {
	type testCase struct {
		Name string
		In   []int16
	}
	cases := []testCase{
		{
			Name: "empty",
			In:   nil,
		},
		{
			Name: "-1, 0, 1",
			In:   []int16{-32768, 0, 32767},
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
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			for _, seek := range []bool{false, true} {
				name := "nonseek"
				if seek {
					name = "seek"
				}
				t.Run(name, func(t *testing.T) {
					var in, out []byte
					if len(c.In) > 0 {
						outF32 := make([]float32, len(c.In))
						for i := range c.In {
							outF32[i] = float32(c.In[i]) / (1 << 15)
						}
						in = unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(c.In))), len(c.In)*2)
						out = unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(outF32))), len(outF32)*4)
					}
					r := convert.NewFloat32BytesReaderFromInt16BytesReader(bytes.NewReader(in)).(io.ReadSeeker)
					var got []byte
					for {
						var buf [97]byte
						n, err := r.Read(buf[:])
						got = append(got, buf[:n]...)
						if err != nil {
							if err != io.EOF {
								t.Fatal(err)
							}
							break
						}
						if seek {
							// Shifting by incomplete bytes should not affect the result.
							for i := range 4 {
								if _, err := r.Seek(int64(i), io.SeekCurrent); err != nil {
									if err != io.EOF {
										t.Fatal(err)
									}
									break
								}
							}
						}
					}
					want := out
					if !bytes.Equal(got, want) {
						t.Errorf("got: %v, want: %v", got, want)
					}
				})
			}
		})
	}
}

// permissiveSeeker is an io.ReadSeeker that silently succeeds without moving for an unrecognized
// whence, so that a rejected seek can only come from the reader under test.
type permissiveSeeker struct {
	r *bytes.Reader
}

func (p *permissiveSeeker) Read(buf []byte) (int, error) {
	return p.r.Read(buf)
}

func (p *permissiveSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart, io.SeekCurrent, io.SeekEnd:
		return p.r.Seek(offset, whence)
	}
	return p.r.Seek(0, io.SeekCurrent)
}

func TestFloat32SeekInvalidWhence(t *testing.T) {
	r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(&permissiveSeeker{r: bytes.NewReader(make([]byte, 256))})

	for _, whence := range []int{-1, 3, 100} {
		if _, err := r.Seek(0, whence); err == nil {
			t.Errorf("Seek(0, %d): got no error, want an error", whence)
		}
	}
}

// dribbleReader is an io.ReadSeeker that returns at most maxN bytes for each Read.
type dribbleReader struct {
	r    *bytes.Reader
	maxN int
}

func (d *dribbleReader) Read(buf []byte) (int, error) {
	if len(buf) > d.maxN {
		buf = buf[:d.maxN]
	}
	return d.r.Read(buf)
}

func (d *dribbleReader) Seek(offset int64, whence int) (int64, error) {
	return d.r.Seek(offset, whence)
}

// float32BytesFromInt16Bytes converts int16 bytes to float32 bytes, ignoring an incomplete sample.
func float32BytesFromInt16Bytes(src []byte) []byte {
	var dst []byte
	for i := range len(src) / 2 {
		v := float32(int16(uint16(src[2*i])|uint16(src[2*i+1])<<8)) / (1 << 15)
		dst = binary.LittleEndian.AppendUint32(dst, math.Float32bits(v))
	}
	return dst
}

func TestFloat32ShortReads(t *testing.T) {
	for _, srcLen := range []int{1, 2, 3, 7, 8, 9} {
		for _, maxN := range []int{1, 2, 3} {
			t.Run(fmt.Sprintf("srcLen=%d,maxN=%d", srcLen, maxN), func(t *testing.T) {
				src := make([]byte, srcLen)
				for i := range src {
					src[i] = byte(i + 1)
				}
				r := convert.NewFloat32BytesReaderFromInt16BytesReader(&dribbleReader{r: bytes.NewReader(src), maxN: maxN})

				var got []byte
				for {
					var buf [64]byte
					n, err := r.Read(buf[:])
					if n == 0 && err == nil {
						t.Fatal("Read: got (0, <nil>), want a non-zero byte count or an error")
					}
					got = append(got, buf[:n]...)
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}

				want := float32BytesFromInt16Bytes(src)
				if !bytes.Equal(got, want) {
					t.Errorf("got: %v, want: %v", got, want)
				}
			})
		}
	}
}

func TestFloat32ShortBuffer(t *testing.T) {
	r := convert.NewFloat32BytesReaderFromInt16BytesReader(bytes.NewReader(make([]byte, 16)))

	if n, err := r.Read(nil); n != 0 || err != nil {
		t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
	}
	for _, l := range []int{1, 2, 3} {
		if n, err := r.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
			t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
		}
	}
}

func TestFloat32SeekEndUnalignedSource(t *testing.T) {
	// The source length is not a multiple of the sample size, so the source ends in the middle of a
	// sample. Every seek from the end must still land on a sample boundary.
	for _, srcLen := range []int{1, 3, 5, 7, 65} {
		t.Run(fmt.Sprintf("srcLen=%d", srcLen), func(t *testing.T) {
			src := make([]byte, srcLen)
			for i := range src {
				src[i] = byte(i + 1)
			}
			// An incomplete sample at the end of the source is not readable, so the whole stream is
			// the one converted from the source truncated to whole samples.
			want := float32BytesFromInt16Bytes(src)

			for offset := -int64(len(want)) - 4; offset <= 4; offset++ {
				r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(bytes.NewReader(src))
				pos, err := r.Seek(offset, io.SeekEnd)
				wantPos := int64(len(want)) + offset/4*4
				if wantPos < 0 {
					if err == nil {
						t.Errorf("Seek(%d, io.SeekEnd): got no error, want an error", offset)
					}
					continue
				}
				if err != nil {
					t.Errorf("Seek(%d, io.SeekEnd): %v", offset, err)
					continue
				}
				if pos != wantPos {
					t.Errorf("Seek(%d, io.SeekEnd): got %d, want %d", offset, pos, wantPos)
					continue
				}
				got, err := io.ReadAll(r)
				if err != nil {
					t.Errorf("reading after Seek(%d, io.SeekEnd): %v", offset, err)
					continue
				}
				if w := want[min(wantPos, int64(len(want))):]; !bytes.Equal(got, w) {
					t.Errorf("reading after Seek(%d, io.SeekEnd): got % x, want % x", offset, got, w)
				}
			}
		})
	}
}

func TestFloat32SeekCurrentAfterPartialSampleRead(t *testing.T) {
	// The source returns 3 bytes at most, which is larger than the sample size but not a multiple of
	// it, so a read leaves a one-byte remainder in the internal buffer and the source position is
	// ahead of the logical position by 1.
	src := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	want := float32BytesFromInt16Bytes(src)

	r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(&dribbleReader{r: bytes.NewReader(src), maxN: 3})

	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got, w := buf[:n], want[:4]; !bytes.Equal(got, w) {
		t.Fatalf("Read: got % x, want % x", got, w)
	}

	// Seek with io.SeekCurrent must be relative to the logical position, so this must be a no-op.
	pos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if w := int64(4); pos != w {
		t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", pos, w)
	}

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want[4:]) {
		t.Errorf("reading after Seek(0, io.SeekCurrent): got % x, want % x", got, want[4:])
	}
}
