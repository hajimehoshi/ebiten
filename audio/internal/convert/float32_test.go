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
						out = unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(outF32))), len(outF32)/2*8)
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
	for i := range len(src) / 4 * 2 {
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
	for _, l := range []int{1, 2, 3, 4, 5, 6, 7} {
		if n, err := r.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
			t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
		}
	}
}

func TestFloat32SeekEndUnalignedSource(t *testing.T) {
	// The source length is not a multiple of the sample size, so the source ends in the middle of a
	// sample. Every seek from the end must still land on a sample boundary.
	for _, srcLen := range []int{1, 2, 3, 5, 6, 7, 65, 66} {
		t.Run(fmt.Sprintf("srcLen=%d", srcLen), func(t *testing.T) {
			src := make([]byte, srcLen)
			for i := range src {
				src[i] = byte(i + 1)
			}
			// An incomplete sample at the end of the source is not readable, so the whole stream is
			// the one converted from the source truncated to whole samples.
			want := float32BytesFromInt16Bytes(src)

			for offset := -int64(len(want)) - 8; offset <= 8; offset++ {
				r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(bytes.NewReader(src))
				pos, err := r.Seek(offset, io.SeekEnd)
				// The requested position, not the one rounded toward the sample boundary,
				// decides whether the seek resolves before the start.
				if int64(len(want))+offset < 0 {
					if err == nil {
						t.Errorf("Seek(%d, io.SeekEnd): got no error, want an error", offset)
					}
					continue
				}
				wantPos := (int64(len(want)) + offset) / 8 * 8
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
	src := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	want := float32BytesFromInt16Bytes(src)

	r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(&dribbleReader{r: bytes.NewReader(src), maxN: 6})

	buf := make([]byte, 16)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got, w := buf[:n], want[:8]; !bytes.Equal(got, w) {
		t.Fatalf("Read: got % x, want % x", got, w)
	}

	pos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if w := int64(8); pos != w {
		t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", pos, w)
	}

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want[8:]) {
		t.Errorf("reading after Seek(0, io.SeekCurrent): got % x, want % x", got, want[8:])
	}
}

// seekCountingReader is an io.ReadSeeker that counts the seeks other than a query for the current position.
type seekCountingReader struct {
	r     io.ReadSeeker
	seeks int
}

func (s *seekCountingReader) Read(buf []byte) (int, error) {
	return s.r.Read(buf)
}

func (s *seekCountingReader) Seek(offset int64, whence int) (int64, error) {
	if offset != 0 || whence != io.SeekCurrent {
		s.seeks++
	}
	return s.r.Seek(offset, whence)
}

func TestSeekCurrentAfterShortBufferDoesNotSeekSource(t *testing.T) {
	for _, tc := range []struct {
		name           string
		newReader      func(io.ReadSeeker) io.ReadSeeker
		bytesPerSample int
	}{
		{
			name: "StereoI16",
			newReader: func(r io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoI16ReadSeeker(r, false, convert.FormatS16)
			},
			bytesPerSample: 4,
		},
		{
			name: "StereoI16Mono",
			newReader: func(r io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoI16ReadSeeker(r, true, convert.FormatU8)
			},
			bytesPerSample: 4,
		},
		{
			name: "StereoF32",
			newReader: func(r io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoF32(r, false)
			},
			bytesPerSample: 8,
		},
		{
			name: "StereoF32Mono",
			newReader: func(r io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoF32(r, true)
			},
			bytesPerSample: 8,
		},
		{
			name:           "Float32",
			newReader:      convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker,
			bytesPerSample: 8,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := randBytes(256)
			want := readAligned(t, tc.newReader(bytes.NewReader(src)))

			r := &seekCountingReader{
				r: bytes.NewReader(src),
			}
			s := tc.newReader(r)
			got := make([]byte, 4*tc.bytesPerSample)
			if _, err := io.ReadFull(s, got); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Read(make([]byte, 1)); !errors.Is(err, io.ErrShortBuffer) {
				t.Fatalf("Read(a buffer of 1 byte): got %v, want %v", err, io.ErrShortBuffer)
			}

			pos, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if pos != int64(len(got)) {
				t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", pos, len(got))
			}
			if r.seeks != 0 {
				t.Errorf("Seek(0, io.SeekCurrent) sought the source %d times, want 0", r.seeks)
			}
			got = append(got, readAligned(t, s)...)
			if !bytes.Equal(got, want) {
				t.Errorf("reading after Seek(0, io.SeekCurrent): got % x, want % x", got, want)
			}

			pos, err = s.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if pos != int64(len(want)) {
				t.Errorf("Seek(0, io.SeekCurrent) at the end: got %d, want %d", pos, len(want))
			}
			if n, err := s.Read(make([]byte, 64)); n != 0 || !errors.Is(err, io.EOF) {
				t.Errorf("Read after Seek(0, io.SeekCurrent) at the end: got (%d, %v), want (0, %v)", n, err, io.EOF)
			}
			if r.seeks != 0 {
				t.Errorf("Seek(0, io.SeekCurrent) at the end sought the source %d times, want 0", r.seeks)
			}
		})
	}
}

// boundedSeeker is an io.ReadSeeker that rejects a seek resolving outside the source, like a real
// audio source does, while a bytes.Reader allows seeking past the end.
type boundedSeeker struct {
	r    io.ReadSeeker
	size int64
}

func (b *boundedSeeker) Read(buf []byte) (int, error) {
	return b.r.Read(buf)
}

func (b *boundedSeeker) Seek(offset int64, whence int) (int64, error) {
	var pos int64
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		cur, err := b.r.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		pos = cur + offset
	case io.SeekEnd:
		pos = b.size + offset
	default:
		return 0, fmt.Errorf("convert_test: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	if pos < 0 || pos > b.size {
		return 0, fmt.Errorf("convert_test: position must be in [0, %d] but was %d", b.size, pos)
	}
	return b.r.Seek(pos, io.SeekStart)
}

func TestFloat32SeekOutOfRangeLeavesStreamIntact(t *testing.T) {
	const srcLen = 200
	// The stream is float32 (8 bytes per sample) converted from int16 (4 bytes per sample).
	const streamLen = srcLen / 2 * 4
	// pos is a sample boundary inside the stream.
	const pos = 40

	src := randBytes(srcLen)
	whole := float32BytesFromInt16Bytes(src)

	for _, tc := range []struct {
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
			offset: -streamLen - 4,
			whence: io.SeekEnd,
		},
		{
			name:   "SeekEndPastEnd",
			offset: 8,
			whence: io.SeekEnd,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// maxN is not a multiple of the sample size, so an incomplete sample remains
			// buffered after a Read.
			r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(&boundedSeeker{r: &dribbleReader{r: bytes.NewReader(src), maxN: 3}, size: srcLen})

			if _, err := r.Seek(pos, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			n, err := r.Read(make([]byte, 64))
			if err != nil {
				t.Fatal(err)
			}
			want := int64(pos + n)

			if _, err := r.Seek(tc.offset, tc.whence); err == nil {
				t.Errorf("Seek(%d, %d): got no error, want an error", tc.offset, tc.whence)
			}

			cur, err := r.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if cur != want {
				t.Errorf("Seek(0, io.SeekCurrent) after a rejected Seek(%d, %d): got %d, want %d", tc.offset, tc.whence, cur, want)
			}

			got := make([]byte, 16)
			if _, err := io.ReadFull(r, got); err != nil {
				t.Fatal(err)
			}
			if w := whole[want : want+int64(len(got))]; !bytes.Equal(got, w) {
				t.Errorf("reading after a rejected Seek(%d, %d): got % x, want % x", tc.offset, tc.whence, got, w)
			}
		})
	}
}

func TestFloat32SeekSmallNegativePosition(t *testing.T) {
	const srcLen = 200

	// The stream is float32 (8 bytes per sample) converted from int16 (4 bytes per sample).
	const streamLen = srcLen / 2 * 4

	src := randBytes(srcLen)
	r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(&boundedSeeker{r: bytes.NewReader(src), size: srcLen})

	// An offset in (-8, 0) is rounded toward the sample boundary, so the requested position must
	// be resolved and checked before the rounding, whichever whence it comes from.
	for _, offset := range []int64{-1, -2, -3, -4, -5, -6, -7, -8} {
		if _, err := r.Seek(offset, io.SeekStart); err == nil {
			t.Errorf("Seek(%d, io.SeekStart): got no error, want an error", offset)
		}
		if _, err := r.Seek(offset, io.SeekCurrent); err == nil {
			t.Errorf("Seek(%d, io.SeekCurrent) at 0: got no error, want an error", offset)
		}
		if _, err := r.Seek(-streamLen+offset, io.SeekEnd); err == nil {
			t.Errorf("Seek(-%d%+d, io.SeekEnd): got no error, want an error", streamLen, offset)
		}
	}

	// The stream must not be broken by the rejected seeks.
	if pos, err := r.Seek(0, io.SeekCurrent); err != nil {
		t.Fatal(err)
	} else if pos != 0 {
		t.Errorf("Seek(0, io.SeekCurrent): got: %d, want: 0", pos)
	}
}

func TestFloat32SourceErrorWithData(t *testing.T) {
	src := make([]byte, 40)
	for i := range src {
		src[i] = byte(i + 1)
	}
	r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(&dataWithErrorReadSeeker{
		src:    bytes.NewReader(src),
		failAt: 1,
		dataN:  3*4 + 1,
	})
	want := float32BytesFromInt16Bytes(src)

	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if !errors.Is(err, errSourceRead) {
		t.Errorf("Read: got error %v, want %v", err, errSourceRead)
	}
	if got, want := n, 3*8; got != want {
		t.Errorf("Read: got %d bytes, want %d", got, want)
	}
	if got, want := buf[:n], want[:n]; !bytes.Equal(got, want) {
		t.Errorf("Read: got %v, want %v", got, want)
	}

	pos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pos, int64(n); got != want {
		t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", got, want)
	}

	rest, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if got := append(buf[:n:n], rest...); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShortBufferEOFAndPosition(t *testing.T) {
	for _, tc := range []struct {
		name      string
		frameSize int
		newReader func(io.ReadSeeker) io.ReadSeeker
	}{
		{
			name:      "Float32",
			frameSize: 8,
			newReader: convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker,
		},
		{
			name:      "StereoI16",
			frameSize: 4,
			newReader: func(src io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoI16ReadSeeker(src, true, convert.FormatS16)
			},
		},
		{
			name:      "StereoF32Mono",
			frameSize: 8,
			newReader: func(src io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoF32(src, true)
			},
		},
		{
			name:      "StereoF32Stereo",
			frameSize: 8,
			newReader: func(src io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoF32(src, false)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte{1, 2, 3, 4, 5, 6, 7, 8}
			want, err := io.ReadAll(tc.newReader(bytes.NewReader(src)))
			if err != nil {
				t.Fatal(err)
			}
			for _, data := range [][]byte{nil, src} {
				r := tc.newReader(bytes.NewReader(data))
				if _, err := r.Seek(0, io.SeekEnd); err != nil {
					t.Fatal(err)
				}
				for size := 1; size < tc.frameSize; size++ {
					if n, err := r.Read(make([]byte, size)); n != 0 || !errors.Is(err, io.EOF) {
						t.Errorf("Read(%d bytes) at EOF = (%d, %v), want (0, EOF)", size, n, err)
					}
				}
			}
			r := tc.newReader(bytes.NewReader(src))
			if n, err := r.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
				t.Errorf("short Read = (%d, %v), want (0, ErrShortBuffer)", n, err)
			}
			if pos, err := r.Seek(0, io.SeekCurrent); err != nil || pos != 0 {
				t.Errorf("current = (%d, %v), want (0, nil)", pos, err)
			}
			r = tc.newReader(bytes.NewReader(src))
			for range 2 {
				if n, err := r.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
					t.Errorf("repeated short Read = (%d, %v)", n, err)
				}
			}

			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("ReadAll after short Read = %x, want %x", got, want)
			}
			r = tc.newReader(&dataWithErrorReadSeeker{
				src:    bytes.NewReader(src),
				failAt: 1,
				dataN:  4,
			})
			if n, err := r.Read(make([]byte, 1)); n != 0 || !errors.Is(err, errSourceRead) {
				t.Errorf("short Read with source error = (%d, %v)", n, err)
			}
			got, err = io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("ReadAll after source error = %x, want %x", got, want)
			}

		})
	}
}

func TestSeekOverflow(t *testing.T) {
	src := make([]byte, 1024)
	for i := range src {
		src[i] = byte(i % 32)
	}
	t.Run("Float32", func(t *testing.T) {
		checkSeekOverflow(t, convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(bytes.NewReader(src)))
	})
	for _, depth := range []int{2, 4} {
		t.Run(fmt.Sprintf("Resampling/depth=%d", depth), func(t *testing.T) {
			checkSeekOverflow(t, convert.NewResampling(bytes.NewReader(src), int64(len(src)), 44100, 48000, depth))
		})
	}
	for _, mono := range []bool{false, true} {
		t.Run(fmt.Sprintf("StereoF32/mono=%t", mono), func(t *testing.T) {
			checkSeekOverflow(t, convert.NewStereoF32(bytes.NewReader(src), mono))
		})
		for _, format := range []convert.Format{convert.FormatU8, convert.FormatS16, convert.FormatS24} {
			t.Run(fmt.Sprintf("StereoI16/mono=%t/format=%d", mono, format), func(t *testing.T) {
				checkSeekOverflow(t, convert.NewStereoI16ReadSeeker(bytes.NewReader(src), mono, format))
			})
		}
	}
}

func checkSeekOverflow(t *testing.T, r io.ReadSeeker) {
	t.Helper()
	want := make([]byte, 128)
	if _, err := io.ReadFull(r, want); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Seek(64, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	for _, whence := range []int{io.SeekCurrent, io.SeekEnd} {
		if _, err := r.Seek(math.MaxInt64, whence); err == nil {
			t.Errorf("Seek(MaxInt64, %d) succeeded", whence)
		}
		if pos, err := r.Seek(0, io.SeekCurrent); err != nil || pos != 64 {
			t.Errorf("position after rejected seek = (%d, %v), want 64", pos, err)
		}
	}
	got := make([]byte, 64)
	if _, err := io.ReadFull(r, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want[64:]) {
		t.Error("audio changed after rejected seeks")
	}
}

func TestStereoSeekEndPositionOverflow(t *testing.T) {
	for _, tc := range []struct {
		name      string
		newReader func(io.ReadSeeker) io.ReadSeeker
	}{
		{
			name: "Float32",
			newReader: func(src io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoF32(src, true)
			},
		},
		{
			name: "Int16",
			newReader: func(src io.ReadSeeker) io.ReadSeeker {
				return convert.NewStereoI16ReadSeeker(src, true, convert.FormatS16)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := io.NewSectionReader(bytes.NewReader(make([]byte, 128)), 0, math.MaxInt64)
			r := tc.newReader(src)
			if _, err := r.Seek(8, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Seek(8, io.SeekEnd); err == nil {
				t.Error("SeekEnd accepted an overflowing output position")
			}
			if pos, err := r.Seek(0, io.SeekCurrent); err != nil || pos != 8 {
				t.Errorf("position after rejected seek = (%d, %v), want 8", pos, err)
			}
			if _, err := io.ReadFull(r, make([]byte, 8)); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSeekEOF(t *testing.T) {
	for _, tc := range []struct {
		name      string
		newReader func(io.ReadSeeker) io.ReadSeeker
	}{
		{
			name:      "Float32",
			newReader: convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker,
		},
		{
			name:      "ResamplingInt16",
			newReader: func(src io.ReadSeeker) io.ReadSeeker { return convert.NewResampling(src, 256, 44100, 48000, 2) },
		},
		{
			name:      "ResamplingFloat32",
			newReader: func(src io.ReadSeeker) io.ReadSeeker { return convert.NewResampling(src, 256, 44100, 48000, 4) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) { checkConvertedEOFSeeks(t, tc.newReader, math.MaxInt64/8*8-1024) })
	}
	for _, mono := range []bool{false, true} {
		t.Run(fmt.Sprintf("StereoF32/mono=%t", mono), func(t *testing.T) {
			checkConvertedEOFSeeks(t, func(src io.ReadSeeker) io.ReadSeeker { return convert.NewStereoF32(src, mono) }, math.MaxInt64/8*8-1024)
		})
		for _, format := range []convert.Format{convert.FormatU8, convert.FormatS16, convert.FormatS24} {
			t.Run(fmt.Sprintf("StereoI16/mono=%t/format=%d", mono, format), func(t *testing.T) {
				maxTarget := int64(math.MaxInt64/8*8 - 1024)
				if !mono && format == convert.FormatS24 {
					maxTarget = math.MaxInt64 / 16 * 8
				}
				checkConvertedEOFSeeks(t, func(src io.ReadSeeker) io.ReadSeeker { return convert.NewStereoI16ReadSeeker(src, mono, format) }, maxTarget)
			})
		}
	}
}

func checkConvertedEOFSeeks(t *testing.T, newReader func(io.ReadSeeker) io.ReadSeeker, maxTarget int64) {
	t.Helper()
	src := make([]byte, 256)
	for i := range src {
		src[i] = byte(i % 32)
	}
	data, err := io.ReadAll(newReader(bytes.NewReader(src)))
	if err != nil {
		t.Fatal(err)
	}
	checkEOFSeeks(t, newReader(bytes.NewReader(src)), int64(len(data)), maxTarget)
}

func checkEOFSeeks(t *testing.T, s io.ReadSeeker, length int64, maxTarget int64) {
	t.Helper()
	want := make([]byte, 64)
	if _, err := io.ReadFull(s, want); err != nil {
		t.Fatal(err)
	}
	for _, target := range []int64{length, length + 32, 1 << 40, maxTarget} {
		for _, whence := range []int{io.SeekStart, io.SeekCurrent, io.SeekEnd} {
			if _, err := s.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			offset := target
			if whence == io.SeekEnd {
				offset -= length
			}
			pos, err := s.Seek(offset, whence)
			if err != nil {
				t.Fatalf("Seek(%d, %d): %v", offset, whence, err)
			}
			if pos != target {
				t.Errorf("Seek(%d, %d) = %d, want %d", offset, whence, pos, target)
			}
			buf := make([]byte, 64)
			for _, size := range []int{1, 3, len(buf), len(buf)} {
				if n, err := s.Read(buf[:size]); n != 0 || !errors.Is(err, io.EOF) {
					t.Errorf("Read at %d = (%d, %v), want (0, EOF)", target, n, err)
				}
			}
			if pos, err := s.Seek(target+1, io.SeekStart); err != nil || pos != target {
				t.Errorf("unaligned seek beyond EOF = (%d, %v), want %d", pos, err, target)
			}
			if pos, err := s.Seek(0, io.SeekCurrent); err != nil || pos != target {
				t.Errorf("current = (%d, %v), want %d", pos, err, target)
			}
			if pos, err := s.Seek(8, io.SeekCurrent); err != nil || pos != target+8 {
				t.Errorf("advance = (%d, %v), want %d", pos, err, target+8)
			}
			if pos, err := s.Seek(-target-8, io.SeekCurrent); err != nil || pos != 0 {
				t.Fatalf("seek back = (%d, %v)", pos, err)
			}
			if _, err := io.ReadFull(s, buf); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(buf, want) {
				t.Error("reading after seeking back returned different bytes")
			}
		}
	}
}

func readWithSizes(t *testing.T, r io.Reader, sizes []int, bytesPerSample int) []byte {
	t.Helper()
	var got []byte
	for i := 0; ; i++ {
		size := sizes[i%len(sizes)]
		buf := make([]byte, size)
		n, err := r.Read(buf)
		if n%bytesPerSample != 0 {
			t.Errorf("Read(a buffer of %d bytes) at %d: got %d bytes, want a multiple of %d", size, len(got), n, bytesPerSample)
		}
		got = append(got, buf[:n]...)
		if errors.Is(err, io.EOF) {
			return got
		}
		if size < bytesPerSample && errors.Is(err, io.ErrShortBuffer) {
			continue
		}
		if err != nil {
			t.Fatalf("Read(a buffer of %d bytes) at %d: %v", size, len(got), err)
		}
		if n == 0 {
			t.Fatalf("Read(a buffer of %d bytes) at %d: got (0, <nil>)", size, len(got))
		}
	}
}

func readAligned(t *testing.T, r io.Reader) []byte {
	t.Helper()
	var got []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		got = append(got, buf[:n]...)
		if errors.Is(err, io.EOF) {
			return got
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadWholeSamples(t *testing.T) {
	const srcLen = 1003
	src := make([]byte, srcLen)
	for i := range src {
		src[i] = byte(i*7 + 1)
	}

	type testCase struct {
		name           string
		bytesPerSample int
		newReader      func(io.ReadSeeker) io.Reader
	}
	cases := []testCase{
		{
			name:           "Float32",
			bytesPerSample: 8,
			newReader: func(src io.ReadSeeker) io.Reader {
				return convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(src)
			},
		},
		{
			name:           "ResamplingInt16",
			bytesPerSample: 4,
			newReader: func(src io.ReadSeeker) io.Reader {
				return convert.NewResampling(src, srcLen, 44100, 48000, 2)
			},
		},
		{
			name:           "ResamplingFloat32",
			bytesPerSample: 8,
			newReader: func(src io.ReadSeeker) io.Reader {
				return convert.NewResampling(src, srcLen, 44100, 48000, 4)
			},
		},
	}
	for _, mono := range []bool{false, true} {
		cases = append(cases, testCase{
			name:           fmt.Sprintf("StereoF32/mono=%t", mono),
			bytesPerSample: 8,
			newReader: func(src io.ReadSeeker) io.Reader {
				return convert.NewStereoF32(src, mono)
			},
		})
		for _, format := range []convert.Format{convert.FormatU8, convert.FormatS16, convert.FormatS24} {
			cases = append(cases, testCase{
				name:           fmt.Sprintf("StereoI16/mono=%t/format=%d", mono, format),
				bytesPerSample: 4,
				newReader: func(src io.ReadSeeker) io.Reader {
					return convert.NewStereoI16ReadSeeker(src, mono, format)
				},
			})
		}
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := readAligned(t, c.newReader(bytes.NewReader(src)))
			for _, maxN := range []int{1, srcLen} {
				t.Run(fmt.Sprintf("maxN=%d", maxN), func(t *testing.T) {
					r := c.newReader(&dribbleReader{
						r:    bytes.NewReader(src),
						maxN: maxN,
					})
					got := readWithSizes(t, r, []int{1, 5, 3, 6, 2, 7, 9, 13, 4, 10, 8, 1027}, c.bytesPerSample)
					if !bytes.Equal(got, want) {
						t.Errorf("got %d bytes, want %d bytes equal to reading with aligned buffers", len(got), len(want))
					}
				})
			}
		})
	}
}
