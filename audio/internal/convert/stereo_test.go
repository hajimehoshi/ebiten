// Copyright 2024 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package convert

import (
	"bytes"
	"io"
	"math"
	"reflect"
	"testing"
)

type testCase struct {
	resolution int
	mono       bool
	input      []byte
	expected   []byte
}

func freq(n int) float64 {
	return float64(n) * math.Pi / 2
}

func writeWord(w *bytes.Buffer, bits int, v uint32) {
	v = v >> (32 - bits)
	for i := 0; i < bits/8; i++ {
		b := byte(v & 0xFF)
		w.WriteByte(b)
		v = v >> 8
	}
}

func genSin(sinLen int, bits int, mono bool) []byte {
	w := bytes.NewBuffer(nil)
	for i := 0; i < sinLen; i++ {
		v := uint32(float64((1<<31)-1) * (1 + math.Sin(freq(i))))
		writeWord(w, bits, v)
		if !mono {
			writeWord(w, bits, v)
		}
	}
	return w.Bytes()
}

func genTestCases() []testCase {
	res16 := genSin(4, 16, false)
	res8 := []byte{}
	for i, v := range res16 {
		if i%2 == 0 {
			v = 0
		}
		res8 = append(res8, v)

	}

	cases := []testCase{
		{8, true, nil, res8},
		{8, false, nil, res8},
		{16, true, nil, res16},
		{16, false, nil, res16},
		{24, true, nil, res16},
		{24, false, nil, res16},
	}
	for i, test := range cases {
		cases[i].input = genSin(10, test.resolution, test.mono)
	}
	return cases
}

func TestIsValidResolution(t *testing.T) {
	tests := []struct {
		v int
		e bool
	}{
		{0, false},
		{1, false},
		{8, true},
		{16, true},
		{24, true},
		{32, false},
	}

	for _, test := range tests {
		if e := IsValidResolution(test.v); e != test.e {
			t.Errorf("IsValidResolution(%d) = %t, expected %t", test.v, e, test.e)
		}
	}
}

func TestDecode(t *testing.T) {
	tests := genTestCases()
	for i, test := range tests {
		s := NewStereo(bytes.NewReader(test.input), test.mono, test.resolution)
		b := make([]byte, len(test.expected))
		_, err := s.Read(b)
		if err != nil {
			t.Fatalf("[%d] Stereo.Read() returned an error: %v", i, err)
		}
		if !reflect.DeepEqual(b, test.expected) {
			t.Errorf(`[%d] Stereo.Read
         input     %v
         result    %v
         expected  %v`, i, test.input, b, test.expected)
		}
	}
}

func TestSeek(t *testing.T) {
	tests := genTestCases()
	for amt := 0; amt < 10; amt += 2 {
		for i, test := range tests {
			s := NewStereo(bytes.NewReader(test.input), test.mono, test.resolution)
			n, err := s.Seek(int64(amt), io.SeekStart)
			v := amt * test.resolution / 8
			if test.mono {
				v /= 2
			}
			if err != nil {
				t.Fatalf("[%d] Stereo.Seek() returned an error: %v", i, err)
			}
			if v != int(n) {
				t.Logf("[%d] Stere.Seek() = %d, expected %d", i, n, v)
			}

			test.expected = test.expected[amt*2:]
			b := make([]byte, len(test.expected))
			_, err = s.Read(b)
			if err != nil {
				t.Fatalf("[%d] Stereo.Read() returned an error: %v", i, err)
			}
			if !reflect.DeepEqual(b, test.expected) {
				t.Errorf(`[%d] Stereo.Read
         input     %v
         result    %v
         expected  %v`, i, test.input, b, test.expected)
			}
		}
	}
}
