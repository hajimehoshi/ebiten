/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shader

import (
	"github.com/go-gl/gl"
	"sync"
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

var once sync.Once

type Matrix interface {
	Element(i, j int) float64
}

// TODO: Use VBO
func DrawTexture(native gl.Texture, target gl.Texture, projectionMatrix [4][4]float64, quads []TextureQuad, geo Matrix, color Matrix) {
	once.Do(func() {
		initialize()
	})

	if len(quads) == 0 {
		return
	}
	// TODO: Check performance
	program := useProgramColorMatrix(glMatrix(projectionMatrix), geo, color)

	gl.ActiveTexture(gl.TEXTURE0)
	native.Bind(gl.TEXTURE_2D)

	if 0 < target {
		gl.ActiveTexture(gl.TEXTURE1)
		target.Bind(gl.TEXTURE_2D)
	}

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

	vertices := []float32{}
	texCoords := []float32{}
	indicies := []uint32{}
	// TODO: Check len(quads) and gl.MAX_ELEMENTS_INDICES?
	for i, quad := range quads {
		x0 := quad.VertexX0
		x1 := quad.VertexX1
		y0 := quad.VertexY0
		y1 := quad.VertexY1
		vertices = append(vertices,
			x0, y0,
			x1, y0,
			x0, y1,
			x1, y1,
		)
		u0 := quad.TextureCoordU0
		u1 := quad.TextureCoordU1
		v0 := quad.TextureCoordV0
		v1 := quad.TextureCoordV1
		texCoords = append(texCoords,
			u0, v0,
			u1, v0,
			u0, v1,
			u1, v1,
		)
		base := uint32(i * 4)
		indicies = append(indicies,
			base, base+1, base+2,
			base+1, base+2, base+3,
		)
	}
	vertexAttrLocation.AttribPointer(2, gl.FLOAT, false, 0, vertices)
	texCoordAttrLocation.AttribPointer(2, gl.FLOAT, false, 0, texCoords)
	gl.DrawElements(gl.TRIANGLES, len(indicies), gl.UNSIGNED_INT, indicies)

	gl.Flush()
}
