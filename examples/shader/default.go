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

// viewportSize is a predefined function.

func Vertex(position vec2, texCoord vec2, color vec4) vec4 {
	return mat4(
		2/viewportSize().x, 0, 0, 0,
		0, 2/viewportSize().y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	) * vec4(position, 0, 1)
}

func Fragment(position vec4) vec4 {
	pos := position.xy/viewportSize() + Cursor/viewportSize()/4
	color := 0.0
	color += sin(pos.x*cos(Time/15)*80) + cos(pos.y*cos(Time/15)*10)
	color += sin(pos.y*sin(Time/10)*40) + cos(pos.x*sin(Time/25)*40)
	color += sin(pos.x*sin(Time/5)*10) + sin(pos.y*sin(Time/35)*80)
	color *= sin(Time/10) * 0.5
	return vec4(color, color*0.5, sin(color+Time/3)*0.75, 1)
}
