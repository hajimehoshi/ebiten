// Copyright 2016 The Ebiten Authors
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

// Element returns a value of a matrix at (i, j).
func (c *ColorM) Element(i, j int) float64 {
	if c.values == "" {
		if i == j {
			return 1
		}
		return 0
	}
	a := js.NewArrayBuffer([]uint8(c.values))
	af64 := js.Global.Get("Float64Array").New(a)
	return af64.Index(i*ColorMDim + j).Float()
}

// SetElement sets an element at (i, j).
func (c *ColorM) SetElement(i, j int, value float64) {
	if c.values == "" {
		c.values = colorMIdentityValue
	}
	a := js.NewArrayBuffer([]uint8(c.values))
	a8 := js.Global.Get("Uint8Array").New(a)
	af64 := js.Global.Get("Float64Array").New(a)
	af64.SetIndex(i*ColorMDim+j, value)
	c.values = string(a8.Interface().([]uint8))
}
