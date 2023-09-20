// Copyright 2020 The Ebiten Authors
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

//go:build ignore

//kage:unit pixels

package main

var Time float
var Cursor vec2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	pos := dstPos.xy - imageDstOrigin()

	dir := normalize(pos - Cursor)
	clr := imageSrc2UnsafeAt(srcPos)

	samples := [...]float{
		-22, -14, -8, -4, -2, 2, 4, 8, 14, 22,
	}
	sum := clr
	for i := 0; i < len(samples); i++ {
		sum += imageSrc2At(srcPos + dir*samples[i])
	}
	sum /= 10 + 1

	dist := distance(pos, Cursor)
	t := clamp(dist/256, 0, 1)
	return mix(clr, sum, t)
}
