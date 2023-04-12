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

package main

var Time float
var Cursor vec2
var ScreenSize vec2

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	srcOrigin, srcSize := imageSrcRegionOnTexture()
	pos := (texCoord - srcOrigin) / srcSize
	pos *= ScreenSize

	dir := normalize(pos - Cursor)
	clr := imageSrc2UnsafeAt(texCoord)

	samples := [...]float{
		-22, -14, -8, -4, -2, 2, 4, 8, 14, 22,
	}
	sum := clr
	for i := 0; i < len(samples); i++ {
		pos := texCoord + dir*samples[i]/imageSrcTextureSize()
		sum += imageSrc2At(pos)
	}
	sum /= 10 + 1

	dist := distance(pos, Cursor)
	t := clamp(dist/256, 0, 1)
	return mix(clr, sum, t)
}
