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
	center := ScreenSize / 2
	// Convert a pixel to a texel by dividing by the texture size.
	// TODO: This is confusing. Add a function to treat pixels (#1431).
	amount := (center - Cursor) / 10 / imageSrcTextureSize()
	var clr vec3
	clr.r = imageSrc2At(texCoord + amount).r
	clr.g = imageSrc2UnsafeAt(texCoord).g
	clr.b = imageSrc2At(texCoord - amount).b
	return vec4(clr, 1.0)
}
