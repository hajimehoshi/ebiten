// Copyright 2014 Hajime Hoshi
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

package internal

import (
	"image/color"
	"math"
)

func NextPowerOf2Int(x int) int {
	x -= 1
	x |= (x >> 1)
	x |= (x >> 2)
	x |= (x >> 4)
	x |= (x >> 8)
	x |= (x >> 16)
	return x + 1
}

func RGBA(clr color.Color) (r, g, b, a float64) {
	cr, cg, cb, ca := clr.RGBA()
	const max = math.MaxUint16
	r = float64(cr) / max
	g = float64(cg) / max
	b = float64(cb) / max
	a = float64(ca) / max
	return
}
