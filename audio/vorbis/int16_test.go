// Copyright 2019 The Ebiten Authors
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

package vorbis_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

type f32reader struct {
	data []float32
	pos  int
}

func (f *f32reader) Read(buf []float32) (int, error) {
	if f.pos == len(f.data) {
		return 0, io.EOF
	}
	n := copy(buf, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func TestInt16BytesReader(t *testing.T) {
	in1 := make([]float32, 256)
	for i := range in1 {
		in1[i] = float32(math.Sin(float64(i)))
	}
	in2 := make([]float32, 65536)
	for i := range in2 {
		in2[i] = float32(math.Cos(float64(i)))
	}

	cases := []struct {
		In       []float32
		Channels int
		N        int
	}{
		{
			In:       in1,
			Channels: 1,
			N:        2,
		},
		{
			In:       in1,
			Channels: 1,
			N:        3,
		},
		{
			In:       in1,
			Channels: 2,
			N:        4,
		},
		{
			In:       in1,
			Channels: 2,
			N:        5,
		},
		{
			In:       in1,
			Channels: 2,
			N:        1024,
		},
		{
			In:       in2,
			Channels: 1,
			N:        2,
		},
		{
			In:       in2,
			Channels: 2,
			N:        4096,
		},
	}

	for i, c := range cases {
		r := vorbis.NewInt16BytesReaderFromFloat32Reader(&f32reader{data: c.In}, c.Channels)

		got := []byte{}
		for {
			buf := make([]byte, c.N)
			n, err := r.Read(buf)
			if err != nil {
				if n == 0 && err == io.EOF {
					break
				}
				t.Fatal(err)
			}
			got = append(got, buf[:n]...)
		}

		want := make([]byte, len(c.In)*2)
		for i, f := range c.In {
			s := int16(f * (1<<15 - 1))
			want[2*i] = byte(s)
			want[2*i+1] = byte(s >> 8)
		}

		if !bytes.Equal(got, want) {
			t.Errorf("case: %d, got: %v, want: %v", i, got, want)
		}
	}
}

func TestInt16BytesReaderShortBuffer(t *testing.T) {
	in := make([]float32, 256)
	for i := range in {
		in[i] = float32(math.Sin(float64(i)))
	}

	cases := []struct {
		Name     string
		Channels int
		Lens     []int
	}{
		{
			Name:     "Mono",
			Channels: 1,
			Lens:     []int{1},
		},
		{
			Name:     "Stereo",
			Channels: 2,
			Lens:     []int{1, 2, 3},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			r := vorbis.NewInt16BytesReaderFromFloat32Reader(&f32reader{data: in}, c.Channels)

			if n, err := r.Read(nil); n != 0 || err != nil {
				t.Errorf("Read(nil): got (%d, %v), want (0, <nil>)", n, err)
			}
			for _, l := range c.Lens {
				if n, err := r.Read(make([]byte, l)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
					t.Errorf("Read(a buffer of %d bytes): got (%d, %v), want (0, %v)", l, n, err, io.ErrShortBuffer)
				}
			}

			// The rejected reads must not consume the source.
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if want := len(in) * 2; len(got) != want {
				t.Errorf("the reader delivered %d bytes, want %d", len(got), want)
			}
		})
	}
}

// f32eofReader returns io.EOF together with the last values.
type f32eofReader struct {
	data []float32
	pos  int
}

func (f *f32eofReader) Read(buf []float32) (int, error) {
	n := copy(buf, f.data[f.pos:])
	f.pos += n
	if f.pos == len(f.data) {
		return n, io.EOF
	}
	return n, nil
}

func TestInt16BytesReaderEOFWithData(t *testing.T) {
	in := []float32{0.1, -0.2, 0.3, -0.4}

	// A buffer shorter than one frame must be rejected instead of delivering a partial sample,
	// which used to leave the second byte behind and lose it at EOF.
	r := vorbis.NewInt16BytesReaderFromFloat32Reader(&f32eofReader{data: in}, 1)
	if n, err := r.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
		t.Errorf("Read(a buffer of 1 bytes): got (%d, %v), want (0, %v)", n, err, io.ErrShortBuffer)
	}

	// Values delivered together with io.EOF must not be dropped.
	var got []byte
	for {
		buf := make([]byte, 2)
		n, err := r.Read(buf)
		got = append(got, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	want := make([]byte, len(in)*2)
	for i, f := range in {
		s := int16(f * (1<<15 - 1))
		want[2*i] = byte(s)
		want[2*i+1] = byte(s >> 8)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
