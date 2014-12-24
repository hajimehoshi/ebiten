// Copyright 2014 Hajime Hoshi
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

package opengl

import (
	"github.com/go-gl/gl"
)

type Filter int

const (
	FilterNearest Filter = gl.NEAREST
	FilterLinear         = gl.LINEAR
)

func Init() {
	gl.Init()
	gl.Enable(gl.TEXTURE_2D)
	// Textures' pixel formats are alpha premultiplied.
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
}

func Clear(r, g, b, a float64) {
	gl.ClearColor(gl.GLclampf(r), gl.GLclampf(g), gl.GLclampf(b), gl.GLclampf(a))
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func Flush() {
	gl.Flush()
}
