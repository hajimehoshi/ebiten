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

// +build js

package jsutil_test

import (
	"syscall/js"
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/jsutil"
)

func TestArrayBufferToSlice(t *testing.T) {
	// TODO
}

func TestSliceToTypedArray(t *testing.T) {
	tests := []struct {
		in  interface{}
		out js.Value
	}{
		{
			in:  []int8{1, 2, 3},
			out: js.Global().Get("Int8Array").Call("of", 1, 2, 3),
		},
		{
			in:  []int16{1, 2, 3},
			out: js.Global().Get("Int16Array").Call("of", 1, 2, 3),
		},
		{
			in:  []int32{1, 2, 3},
			out: js.Global().Get("Int32Array").Call("of", 1, 2, 3),
		},
		{
			in:  []uint8{1, 2, 3},
			out: js.Global().Get("Uint8Array").Call("of", 1, 2, 3),
		},
		{
			in:  []uint16{1, 2, 3},
			out: js.Global().Get("Uint16Array").Call("of", 1, 2, 3),
		},
		{
			in:  []uint32{1, 2, 3},
			out: js.Global().Get("Uint32Array").Call("of", 1, 2, 3),
		},
		{
			in:  []float32{1, 2, 3},
			out: js.Global().Get("Float32Array").Call("of", 1, 2, 3),
		},
		{
			in:  []float64{1, 2, 3},
			out: js.Global().Get("Float64Array").Call("of", 1, 2, 3),
		},
	}
	for _, test := range tests {
		got, free := SliceToTypedArray(test.in)
		defer free()
		want := test.out
		if got.Get("constructor") != want.Get("constructor") {
			t.Errorf("class: got: %s, want: %s", got.Get("constructor").Get("name"), want.Get("constructor").Get("name"))
		}
		if got.Length() != want.Length() {
			t.Errorf("length: got: %d, want: %d", got.Length(), want.Length())
		}
		for i := 0; i < got.Length(); i++ {
			gotv := got.Index(i).Float()
			wantv := want.Index(i).Float()
			if gotv != wantv {
				t.Errorf("[%d]: got: %v, want: %v", i, gotv, wantv)
			}
		}
	}
}
