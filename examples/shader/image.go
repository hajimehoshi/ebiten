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

var Image texture2d

func Vertex(position vec2, texCoord vec2, color vec4) (vec4, vec2) {
	return mat4(
		2/viewportSize().x, 0, 0, 0,
		0, 2/viewportSize().y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	) * vec4(position, 0, 1), texCoord
}

func Fragment(position vec4, tex vec2) vec4 {
	// TODO: Instead of using texture2D directly, define and use special functions for Ebiten images.
	return texture2D(Image, tex)
}
