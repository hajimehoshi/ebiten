package graphics

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"image/color"
	"math"
	"unsafe"
)

type GraphicsContext struct {
	screenWidth int
	screenHeight int
	screenScale int
	mainFramebuffer C.GLuint
	projectionMatrix [16]float32
	currentShaderProgram C.GLuint
	framebuffers map[C.GLuint]C.GLuint
}

// This method should be called on the UI thread.
func newGraphicsContext(screenWidth, screenHeight, screenScale int) *GraphicsContext {
	context := &GraphicsContext{
		screenWidth: screenWidth,
		screenHeight: screenHeight,
		screenScale: screenScale,
		mainFramebuffer: 0,
		framebuffers: map[C.GLuint]C.GLuint{},
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

func (context *GraphicsContext) Fill(color color.Color) {
	r, g, b, a := color.RGBA()
	max := 65535.0
	C.glClearColor(
		C.GLclampf(float64(r) / max),
		C.GLclampf(float64(g) / max),
		C.GLclampf(float64(b) / max),
		C.GLclampf(float64(a) / max))
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (context *GraphicsContext) DrawRect(x, y, width, height int, color color.Color) {
	// TODO: implement!
}

func (context *GraphicsContext) DrawTexture(texture *Texture,
	srcX, srcY, srcWidth, srcHeight int,
	geometryMatrix *GeometryMatrix, colorMatrix *ColorMatrix) {
	geometryMatrix = geometryMatrix.Clone()
	colorMatrix    = colorMatrix.Clone()

	context.setShaderProgram(geometryMatrix, colorMatrix)
	C.glBindTexture(C.GL_TEXTURE_2D, texture.id)

	x1 := float32(0)
	x2 := float32(srcWidth)
	y1 := float32(0)
	y2 := float32(srcHeight)
	vertex := [...]float32{
		x1, y1,
		x2, y1,
		x1, y2,
		x2, y2,
	}

	tu1 := float32(srcX)             / float32(texture.TextureWidth)
	tu2 := float32(srcX + srcWidth)  / float32(texture.TextureWidth)
	tv1 := float32(srcY)             / float32(texture.TextureHeight)
	tv2 := float32(srcY + srcHeight) / float32(texture.TextureHeight)
	texCoord := [...]float32{
		tu1, tv1,
		tu2, tv1,
		tu1, tv2,
		tu2, tv2,
	}

	vertexAttrLocation := getAttributeLocation(context.currentShaderProgram, "vertex")
	textureAttrLocation := getAttributeLocation(context.currentShaderProgram, "texture")

	C.glEnableClientState(C.GL_VERTEX_ARRAY)
	C.glEnableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glEnableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glEnableVertexAttribArray(C.GLuint(textureAttrLocation))
	C.glVertexAttribPointer(C.GLuint(vertexAttrLocation), 2, C.GL_FLOAT, C.GL_FALSE,
		0, unsafe.Pointer(&vertex[0]))
	C.glVertexAttribPointer(C.GLuint(textureAttrLocation), 2, C.GL_FLOAT, C.GL_FALSE,
		0, unsafe.Pointer(&texCoord[0]))
	C.glDrawArrays(C.GL_TRIANGLE_STRIP, 0, 4)
	C.glDisableVertexAttribArray(C.GLuint(textureAttrLocation))
	C.glDisableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glDisableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glDisableClientState(C.GL_VERTEX_ARRAY)
}

func (context *GraphicsContext) SetOffscreen(texture *Texture) {
	framebuffer := C.GLuint(0)
	if texture != nil {
		framebuffer = context.getFramebuffer(texture)
	} else {
		framebuffer = context.mainFramebuffer
	}
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) != C.GL_FRAMEBUFFER_COMPLETE {
		panic("glBindFramebuffer failed")
	}
	C.glEnable(C.GL_BLEND)
	C.glBlendFunc(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA)

	width, height, tx, ty := 0, 0, 0, 0
	if framebuffer != context.mainFramebuffer {
		width  = texture.TextureWidth
		height = texture.TextureHeight
		tx     = -1
		ty     = -1
	} else {
		width  = context.screenWidth * context.screenScale
		height = -1 * context.screenHeight * context.screenScale
		tx     = -1
		ty     = 1
	}
	C.glViewport(0, 0,
		C.GLsizei(math.Abs(float64(width))),
		C.GLsizei(math.Abs(float64(height))))
	e11 := float32(2.0) / float32(width)
	e22 := float32(2.0) / float32(height)
	e41 := float32(tx)
	e42 := float32(ty)
	context.projectionMatrix = [...]float32{
		e11, 0,   0, 0,
		0,   e22, 0, 0,
		0,   0,   1, 0,
		e41, e42, 0, 1,
	}
}

func (context *GraphicsContext) resetOffscreen() {
	context.SetOffscreen(nil)
}

// This method should be called on the UI thread.
func (context *GraphicsContext) flush() {
	C.glFlush()
}

// This method should be called on the UI thread.
func (context *GraphicsContext) setShaderProgram(
	geometryMatrix *GeometryMatrix, colorMatrix *ColorMatrix) {
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

	a  := float32(geometryMatrix.A())
	b  := float32(geometryMatrix.B())
	c  := float32(geometryMatrix.C())
	d  := float32(geometryMatrix.D())
	tx := float32(geometryMatrix.Tx())
	ty := float32(geometryMatrix.Ty())
	glModelviewMatrix := [...]float32{
		a,  c,  0, 0,
		b,  d,  0, 0,
		0,  0,  1, 0,
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
			e[i][j] = float32(colorMatrix.Element(i, j))
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

// This method should be called on the UI thread.
func (context *GraphicsContext) getFramebuffer(texture *Texture) C.GLuint{
	framebuffer, ok := context.framebuffers[texture.id]
	if ok {
		return framebuffer
	}

	newFramebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &newFramebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, newFramebuffer)
	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, texture.id, 0)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) != C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	context.framebuffers[texture.id] = newFramebuffer
	return newFramebuffer
}

// This method should be called on the UI thread.
func (context *GraphicsContext) deleteFramebuffer(texture *Texture) {
	framebuffer, ok := context.framebuffers[texture.id]
	if !ok {
		// TODO: panic?
		return
	}
	C.glDeleteFramebuffers(1, &framebuffer)
	delete(context.framebuffers, texture.id)
}
