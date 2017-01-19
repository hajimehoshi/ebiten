// Copyright 2017 The Ebiten Authors
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

package affine

import (
	"github.com/gopherjs/gopherjs/js"
)

func elements(values string, dim int) []float64 {
	if values == "" {
		result := make([]float64, dim*(dim-1))
		for i := 0; i < dim-1; i++ {
			result[i*dim+i] = 1
		}
		return result
	}
	a := js.NewArrayBuffer([]uint8(values))
	return js.Global.Get("Float64Array").New(a).Interface().([]float64)
}

func setElements(values []float64, dim int) string {
	a := js.NewArrayBuffer(make([]uint8, len(values)*8))
	a8 := js.Global.Get("Uint8Array").New(a)
	af64 := js.Global.Get("Float64Array").New(a)
	for i, v := range values {
		af64.SetIndex(i, v)
	}
	return string(a8.Interface().([]uint8))
}

func setElement(values string, dim int, i, j int, value float64) string {
	if values == "" {
		values = identityValues[dim]
	}
	a := js.NewArrayBuffer([]uint8(values))
	a8 := js.Global.Get("Uint8Array").New(a)
	af64 := js.Global.Get("Float64Array").New(a)
	af64.SetIndex(i*dim+j, value)
	return string(a8.Interface().([]uint8))
}
