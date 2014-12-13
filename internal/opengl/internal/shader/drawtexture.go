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
func DrawTexture(native gl.Texture, projectionMatrix [4][4]float64, quads []TextureQuad, geo Matrix, color Matrix) {
	once.Do(func() {
		initialize()
	})

	if len(quads) == 0 {
		return
	}
	// TODO: Check performance
	shaderProgram := use(glMatrix(projectionMatrix), geo, color)

	native.Bind(gl.TEXTURE_2D)
	defer gl.Texture(0).Bind(gl.TEXTURE_2D)

	vertexAttrLocation := getAttributeLocation(shaderProgram, "vertex")
	texCoordAttrLocation := getAttributeLocation(shaderProgram, "tex_coord")

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
		x1 := quad.VertexX1
		x2 := quad.VertexX2
		y1 := quad.VertexY1
		y2 := quad.VertexY2
		vertices = append(vertices,
			x1, y1,
			x2, y1,
			x1, y2,
			x2, y2,
		)
		u1 := quad.TextureCoordU1
		u2 := quad.TextureCoordU2
		v1 := quad.TextureCoordV1
		v2 := quad.TextureCoordV2
		texCoords = append(texCoords,
			u1, v1,
			u2, v1,
			u1, v2,
			u2, v2,
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
}
