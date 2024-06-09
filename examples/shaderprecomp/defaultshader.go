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
	pos := (dstPos.xy - imageDstOrigin()) / imageDstSize()
	pos += Cursor / imageDstSize() / 4
	clr := 0.0
	clr += sin(pos.x*cos(Time/15)*80) + cos(pos.y*cos(Time/15)*10)
	clr += sin(pos.y*sin(Time/10)*40) + cos(pos.x*sin(Time/25)*40)
	clr += sin(pos.x*sin(Time/5)*10) + sin(pos.y*sin(Time/35)*80)
	clr *= sin(Time/10) * 0.5
	return vec4(clr, clr*0.5, sin(clr+Time/3)*0.75, 1)
}
