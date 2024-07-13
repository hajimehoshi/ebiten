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
	"crypto/rand"
	"io"
	"runtime"
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

func randInt16s(n int) []int16 {
	r := make([]int16, n)
	if _, err := rand.Read(unsafe.Slice((*byte)(unsafe.Pointer(&r[0])), len(r)*2)); err != nil {
		panic(err)
	}
	return r
}

func TestFloat32(t *testing.T) {
	cases := []struct {
		In                  []int16
		SkipOnBrowserReason string
	}{
		{
			In: nil,
		},
		{
			In: []int16{-32768, 0, 32767},
		},
		{
			In: []int16{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			In: randInt16s(256),
		},
		{
			In:                  randInt16s(65536),
			SkipOnBrowserReason: "entropy is not enough on browser to generate random numbers",
		},
	}
	for _, c := range cases {
		if runtime.GOOS == "js" && c.SkipOnBrowserReason != "" {
			t.Skip(c.SkipOnBrowserReason)
		}
		// Note that unsafe.SliceData is available as of Go 1.20.
		var in, out []byte
		if len(c.In) > 0 {
			outF32 := make([]float32, len(c.In))
			for i := range c.In {
				outF32[i] = float32(c.In[i]) / (1 << 15)
			}
			in = unsafe.Slice((*byte)(unsafe.Pointer(&c.In[0])), len(c.In)*2)
			out = unsafe.Slice((*byte)(unsafe.Pointer(&outF32[0])), len(outF32)*4)
		}
		r := convert.NewFloat32BytesReaderFromInt16BytesReader(bytes.NewReader(in))
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
		}
		want := out
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
}
