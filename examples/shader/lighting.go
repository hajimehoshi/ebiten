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

// Uniform variables.
var Time float
var Cursor vec2

// Fragment is the entry point of the fragment shader.
// Fragment returns the color value for the current position.
func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// You can define variables with a short variable declaration like Go.
	pos := dstPos.xy - imageDstOrigin()

	lightpos := vec3(Cursor, 50)
	lightdir := normalize(lightpos - vec3(pos, 0))
	normal := normalize(imageSrc1UnsafeAt(srcPos) - 0.5)
	const ambient = 0.25
	diffuse := 0.75 * max(0.0, dot(normal.xyz, lightdir))

	// You can treat multiple source images by
	// imageSrc[N]At or imageSrc[N]UnsafeAt.
	return imageSrc0UnsafeAt(srcPos) * (ambient + diffuse)
}
