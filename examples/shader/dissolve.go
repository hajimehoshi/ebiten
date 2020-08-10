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

// +build ignore

package main

var Time float
var Cursor vec2
var ImageSize vec2

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// Triangle wave to go 0-->1-->0...
	Level := abs(2*fract(Time/3) - 1)

	a := step(Level, image3TextureAt(texCoord).x)
	if image3TextureAt(texCoord).x < Level && image3TextureAt(texCoord).x > Level-0.1 {
		return vec4(1) * image0TextureAt(texCoord).w
	}

	return vec4(a) * image0TextureAt(texCoord)
}
