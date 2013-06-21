package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"fmt"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	"unsafe"
)

type GraphicsContext struct {
	screenWidth          int
	screenHeight         int
	screenScale          int
	textures             map[graphics.TextureID]*Texture
	projectionMatrix     [16]float32
	currentShaderProgram C.GLuint
	mainFramebuffer      C.GLuint
	framebuffers         map[C.GLuint]C.GLuint
}

// This method should be called on the UI thread.
func newGraphicsContext(screenWidth, screenHeight, screenScale int) *GraphicsContext {
	context := &GraphicsContext{
		screenWidth:     screenWidth,
		screenHeight:    screenHeight,
		screenScale:     screenScale,
		textures:        map[graphics.TextureID]*Texture{},
		mainFramebuffer: 0,
		framebuffers:    map[C.GLuint]C.GLuint{},
	}
	// main framebuffer should be created sooner than any other framebuffers!
	mainFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &mainFramebuffer)
	context.mainFramebuffer = C.GLuint(mainFramebuffer)

	initializeShaders()

	return context
}

func (context *GraphicsContext) Clear() {
	C.glClearColor(0, 0, 0, 1)
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (context *GraphicsContext) Fill(clr color.Color) {
	r, g, b, a := clr.RGBA()
	max := 65535.0
	C.glClearColor(
		C.GLclampf(float64(r)/max),
		C.GLclampf(float64(g)/max),
		C.GLclampf(float64(b)/max),
		C.GLclampf(float64(a)/max))
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (context *GraphicsContext) DrawRect(x, y, width, height int, clr color.Color) {
	// TODO: implement!
}

func (context *GraphicsContext) DrawTexture(
	texture graphics.Texture,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	source := graphics.Rect{0, 0, texture.Width, texture.Height}
	locations := []graphics.TextureLocation{{0, 0, source}}
	context.DrawTextureParts(texture.ID, locations,
		geometryMatrix, colorMatrix)
}

func (context *GraphicsContext) DrawTextureParts(
	textureID graphics.TextureID, locations []graphics.TextureLocation,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {

	texture := context.textures[textureID]

	context.setShaderProgram(geometryMatrix, colorMatrix)
	C.glBindTexture(C.GL_TEXTURE_2D, texture.id)

	vertexAttrLocation := getAttributeLocation(context.currentShaderProgram, "vertex")
	textureAttrLocation := getAttributeLocation(context.currentShaderProgram, "texture")

	C.glEnableClientState(C.GL_VERTEX_ARRAY)
	C.glEnableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glEnableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glEnableVertexAttribArray(C.GLuint(textureAttrLocation))
	for _, location := range locations {
		x1 := float32(location.LocationX)
		x2 := float32(location.LocationX + location.Source.Width)
		y1 := float32(location.LocationY)
		y2 := float32(location.LocationY + location.Source.Height)
		vertex := [...]float32{
			x1, y1,
			x2, y1,
			x1, y2,
			x2, y2,
		}

		src := location.Source
		tu1 := float32(src.X) / float32(texture.textureWidth)
		tu2 := float32(src.X+src.Width) / float32(texture.textureWidth)
		tv1 := float32(src.Y) / float32(texture.textureHeight)
		tv2 := float32(src.Y+src.Height) / float32(texture.textureHeight)
		texCoord := [...]float32{
			tu1, tv1,
			tu2, tv1,
			tu1, tv2,
			tu2, tv2,
		}
		C.glVertexAttribPointer(C.GLuint(vertexAttrLocation), 2, C.GL_FLOAT, C.GL_FALSE,
			0, unsafe.Pointer(&vertex[0]))
		C.glVertexAttribPointer(C.GLuint(textureAttrLocation), 2, C.GL_FLOAT, C.GL_FALSE,
			0, unsafe.Pointer(&texCoord[0]))
		C.glDrawArrays(C.GL_TRIANGLE_STRIP, 0, 4)
	}
	C.glDisableVertexAttribArray(C.GLuint(textureAttrLocation))
	C.glDisableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glDisableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glDisableClientState(C.GL_VERTEX_ARRAY)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (context *GraphicsContext) SetOffscreen(textureID graphics.TextureID) {
	texture := context.textures[textureID]
	framebuffer := context.getFramebuffer(texture.id)
	if framebuffer == context.mainFramebuffer {
		panic("invalid framebuffer")
	}
	context.setOffscreenFramebuffer(framebuffer,
		texture.textureWidth, texture.textureHeight)
}

func (context *GraphicsContext) setOffscreenFramebuffer(framebuffer C.GLuint,
	textureWidth, textureHeight int) {
	C.glFlush()

	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	if err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER); err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}
	C.glEnable(C.GL_BLEND)
	C.glBlendFunc(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA)

	width, height, tx, ty := 0, 0, 0, 0
	if framebuffer != context.mainFramebuffer {
		width = textureWidth
		height = textureHeight
		tx = -1
		ty = -1
	} else {
		width = context.screenWidth * context.screenScale
		height = -1 * context.screenHeight * context.screenScale
		tx = -1
		ty = 1
	}
	C.glViewport(0, 0, C.GLsizei(abs(width)), C.GLsizei(abs(height)))
	e11 := float32(2.0) / float32(width)
	e22 := float32(2.0) / float32(height)
	e41 := float32(tx)
	e42 := float32(ty)
	context.projectionMatrix = [...]float32{
		e11, 0, 0, 0,
		0, e22, 0, 0,
		0, 0, 1, 0,
		e41, e42, 0, 1,
	}
}

func (context *GraphicsContext) resetOffscreen() {
	context.setOffscreenFramebuffer(context.mainFramebuffer, 0, 0)
}

// This method should be called on the UI thread.
func (context *GraphicsContext) flush() {
	C.glFlush()
}

// This method should be called on the UI thread.
func (context *GraphicsContext) setShaderProgram(
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	program := C.GLuint(0)
	if colorMatrix.IsIdentity() {
		program = regularShaderProgram
	} else {
		program = colorMatrixShaderProgram
	}
	// TODO: cache and skip?
	C.glUseProgram(program)
	context.currentShaderProgram = program

	C.glUniformMatrix4fv(getUniformLocation(program, "projection_matrix"),
		1, C.GL_FALSE,
		(*C.GLfloat)(&context.projectionMatrix[0]))

	a := float32(geometryMatrix.Elements[0][0])
	b := float32(geometryMatrix.Elements[0][1])
	c := float32(geometryMatrix.Elements[1][0])
	d := float32(geometryMatrix.Elements[1][1])
	tx := float32(geometryMatrix.Elements[0][2])
	ty := float32(geometryMatrix.Elements[1][2])
	glModelviewMatrix := [...]float32{
		a, c, 0, 0,
		b, d, 0, 0,
		0, 0, 1, 0,
		tx, ty, 0, 1,
	}
	C.glUniformMatrix4fv(getUniformLocation(program, "modelview_matrix"),
		1, C.GL_FALSE,
		(*C.GLfloat)(&glModelviewMatrix[0]))

	C.glUniform1i(getUniformLocation(program, "texture"), 0)

	if program != colorMatrixShaderProgram {
		return
	}

	e := [4][5]float32{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			e[i][j] = float32(colorMatrix.Elements[i][j])
		}
	}

	glColorMatrix := [...]float32{
		e[0][0], e[0][1], e[0][2], e[0][3],
		e[1][0], e[1][1], e[1][2], e[1][3],
		e[2][0], e[2][1], e[2][2], e[2][3],
		e[3][0], e[3][1], e[3][2], e[3][3],
	}
	C.glUniformMatrix4fv(getUniformLocation(program, "color_matrix"),
		1, C.GL_FALSE, (*C.GLfloat)(&glColorMatrix[0]))

	glColorMatrixTranslation := [...]float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	C.glUniform4fv(getUniformLocation(program, "color_matrix_translation"),
		1, (*C.GLfloat)(&glColorMatrixTranslation[0]))
}

func (context *GraphicsContext) getFramebuffer(textureID C.GLuint) C.GLuint {
	framebuffer, ok := context.framebuffers[textureID]
	if ok {
		return framebuffer
	}

	newFramebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &newFramebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, newFramebuffer)
	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, textureID, 0)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) != C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	context.framebuffers[textureID] = newFramebuffer
	return newFramebuffer
}

func (context *GraphicsContext) deleteFramebuffer(textureID C.GLuint) {
	framebuffer, ok := context.framebuffers[textureID]
	if !ok {
		// TODO: panic?
		return
	}
	C.glDeleteFramebuffers(1, &framebuffer)
	delete(context.framebuffers, textureID)
}

func (context *GraphicsContext) NewTexture(width, height int) graphics.Texture {
	texture := newTexture(width, height)
	id := graphics.TextureID(texture.id)
	context.textures[id] = texture
	return graphics.Texture{
		ID:     id,
		Width:  texture.width,
		Height: texture.height,
	}
}

func (context *GraphicsContext) NewTextureFromImage(img image.Image) graphics.Texture {
	texture := newTextureFromImage(img)
	id := graphics.TextureID(texture.id)
	context.textures[id] = texture
	return graphics.Texture{
		ID:     id,
		Width:  texture.width,
		Height: texture.height,
	}
}
