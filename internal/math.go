/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internal

import (
	"image/color"
	"math"
)

func NextPowerOf2(x uint64) uint64 {
	x -= 1
	x |= (x >> 1)
	x |= (x >> 2)
	x |= (x >> 4)
	x |= (x >> 8)
	x |= (x >> 16)
	x |= (x >> 32)
	return x + 1
}

func NextPowerOf2Int(size int) int {
	return int(NextPowerOf2(uint64(size)))
}

func RGBA(clr color.Color) (r, g, b, a float64) {
	clr2 := color.NRGBA64Model.Convert(clr).(color.NRGBA64)
	const max = math.MaxUint16
	r = float64(clr2.R) / max
	g = float64(clr2.G) / max
	b = float64(clr2.B) / max
	a = float64(clr2.A) / max
	return
}
