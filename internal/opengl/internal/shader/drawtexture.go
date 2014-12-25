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

package shader

import (
	"github.com/go-gl/gl"
	"unsafe"
)

func glMatrix(m [4][4]float64) [16]float32 {
	result := [16]float32{}
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			result[i+j*4] = float32(m[i][j])
		}
	}
	return result
}

type Matrix interface {
	Element(i, j int) float64
}

var initialized = false

const size = 10000
const uint16Size = 2
const short32Size = 4

func DrawTexture(native gl.Texture, projectionMatrix [4][4]float64, quads []TextureQuad, geo Matrix, color Matrix) {
	// TODO: Check len(quads) and gl.MAX_ELEMENTS_INDICES?
	const stride = 4 * 4
	if !initialized {
		initialize()

		indexBuffer := gl.GenBuffer()
		indexBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
		indices := make([]uint16, 6*size)
		for i := uint16(0); i < size; i++ {
			indices[6*i+0] = 4*i + 0
			indices[6*i+1] = 4*i + 1
			indices[6*i+2] = 4*i + 2
			indices[6*i+3] = 4*i + 1
			indices[6*i+4] = 4*i + 2
			indices[6*i+5] = 4*i + 3
		}
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, uint16Size*len(indices), indices, gl.STATIC_DRAW)

		vertexBuffer := gl.GenBuffer()
		vertexBuffer.Bind(gl.ARRAY_BUFFER)
		vertices := make([]float32, stride*size)
		gl.BufferData(gl.ARRAY_BUFFER, short32Size*len(vertices), vertices, gl.DYNAMIC_DRAW)

		initialized = true
	}

	if len(quads) == 0 {
		return
	}
	// TODO: Check performance
	program := useProgramColorMatrix(glMatrix(projectionMatrix), geo, color)

	gl.ActiveTexture(gl.TEXTURE0)
	native.Bind(gl.TEXTURE_2D)

	vertexAttrLocation := getAttributeLocation(program, "vertex")
	texCoordAttrLocation := getAttributeLocation(program, "tex_coord")

	gl.EnableClientState(gl.VERTEX_ARRAY)
	gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
	vertexAttrLocation.EnableArray()
	texCoordAttrLocation.EnableArray()
	defer func() {
		texCoordAttrLocation.DisableArray()
		vertexAttrLocation.DisableArray()
		gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)
		gl.DisableClientState(gl.VERTEX_ARRAY)
	}()

	// TODO: Fix this dirty hack after fixing https://github.com/go-gl/gl/issues/174
	v := (*int)(unsafe.Pointer(uintptr(4 * 0)))
	t := (*int)(unsafe.Pointer(uintptr(4 * 2)))
	vertexAttrLocation.AttribPointer(2, gl.FLOAT, false, stride, v)
	texCoordAttrLocation.AttribPointer(2, gl.FLOAT, false, stride, t)

	vertices := []float32{}
	for _, quad := range quads {
		x0 := quad.VertexX0
		x1 := quad.VertexX1
		y0 := quad.VertexY0
		y1 := quad.VertexY1
		u0 := quad.TextureCoordU0
		u1 := quad.TextureCoordU1
		v0 := quad.TextureCoordV0
		v1 := quad.TextureCoordV1
		vertices = append(vertices,
			x0, y0, u0, v0,
			x1, y0, u1, v0,
			x0, y1, u0, v1,
			x1, y1, u1, v1,
		)
	}
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, short32Size*len(vertices), vertices)
	// TODO: Pass 0 after fixing the gl bug
	gl.DrawElements(gl.TRIANGLES, 6*len(quads), gl.UNSIGNED_SHORT, nil)

	gl.Flush()
}
