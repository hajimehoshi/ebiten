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
var ScreenSize vec2

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// Triangle wave to go 0-->1-->0...
	limit := abs(2*fract(Time/3) - 1)
	level := imageSrc3UnsafeAt(texCoord).x

	// Add a white border
	if limit-0.1 < level && level < limit {
		alpha := imageSrc0UnsafeAt(texCoord).w
		return vec4(alpha)
	}

	return step(limit, level) * imageSrc0UnsafeAt(texCoord)
}
