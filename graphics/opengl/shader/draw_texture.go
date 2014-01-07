package shader

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
	"sync"
	"unsafe"
)

var once sync.Once

func DrawTexture(native texture.Native, projectionMatrix [16]float32, quads []graphics.TextureQuad, geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	once.Do(func() {
		initialize()
	})

	if len(quads) == 0 {
		return
	}
	shaderProgram := use(projectionMatrix, geometryMatrix, colorMatrix)
	defer C.glUseProgram(0)

	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(native))
	defer C.glBindTexture(C.GL_TEXTURE_2D, 0)

	vertexAttrLocation := getAttributeLocation(shaderProgram, "vertex")
	texCoordAttrLocation := getAttributeLocation(shaderProgram, "tex_coord")

	C.glEnableClientState(C.GL_VERTEX_ARRAY)
	C.glEnableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glEnableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glEnableVertexAttribArray(C.GLuint(texCoordAttrLocation))
	defer func() {
		C.glDisableVertexAttribArray(C.GLuint(texCoordAttrLocation))
		C.glDisableVertexAttribArray(C.GLuint(vertexAttrLocation))
		C.glDisableClientState(C.GL_TEXTURE_COORD_ARRAY)
		C.glDisableClientState(C.GL_VERTEX_ARRAY)
	}()

	vertices := []float32{}
	texCoords := []float32{}
	indicies := []uint32{}
	// TODO: Check len(parts) and GL_MAX_ELEMENTS_INDICES
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
	C.glVertexAttribPointer(C.GLuint(vertexAttrLocation), 2,
		C.GL_FLOAT, C.GL_FALSE,
		0, unsafe.Pointer(&vertices[0]))
	C.glVertexAttribPointer(C.GLuint(texCoordAttrLocation), 2,
		C.GL_FLOAT, C.GL_FALSE,
		0, unsafe.Pointer(&texCoords[0]))
	C.glDrawElements(C.GL_TRIANGLES, C.GLsizei(len(indicies)),
		C.GL_UNSIGNED_INT, unsafe.Pointer(&indicies[0]))
}
