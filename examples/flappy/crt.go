// Copyright 2022 The Ebitengine Authors
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

// Reference: a public domain CRT effect
// https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-lottes.glsl

package main

func Warp(pos vec2) vec2 {
	const (
		warpX = 0.031
		warpY = 0.041
	)
	pos = pos*2 - 1
	pos *= vec2(1+(pos.y*pos.y)*warpX, 1+(pos.x*pos.x)*warpY)
	return pos/2 + 0.5
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// Adjust the texture position to [0, 1].
	origin := imageSrc0Origin()
	size := imageSrc0Size()
	pos := srcPos
	pos -= origin
	pos /= size

	pos = Warp(pos)
	return imageSrc0At(pos*size + origin)
}
