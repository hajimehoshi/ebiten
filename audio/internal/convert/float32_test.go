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

package convert_test

import (
	"bytes"
	"io"
	"math"
	"testing"

	. "github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
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

func newFloat32Reader(data []float32) Float32Reader {
	return &f32reader{data: data}
}

func TestFloat32Reader(t *testing.T) {
	in1 := make([]float32, 256)
	for i := range in1 {
		in1[i] = float32(math.Sin(float64(i)))
	}
	in2 := make([]float32, 65536)
	for i := range in2 {
		in2[i] = float32(math.Cos(float64(i)))
	}

	cases := []struct {
		In []float32
		N  int
	}{
		{
			In: in1,
			N:  1,
		},
		{
			In: in1,
			N:  2,
		},
		{
			In: in1,
			N:  3,
		},
		{
			In: in1,
			N:  1024,
		},
		{
			In: in2,
			N:  1,
		},
		{
			In: in2,
			N:  4096,
		},
	}

	for i, c := range cases {
		r := NewReaderFromFloat32Reader(newFloat32Reader(c.In))

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
