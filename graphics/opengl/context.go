package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
	"math"
	"unsafe"
)

type Context struct {
	screen                 *RenderTarget
	screenWidth            int
	screenHeight           int
	screenScale            int
	textures               map[C.GLuint]*Texture
	currentOffscreen       *RenderTarget
	mainFramebufferTexture *RenderTarget
}

func newContext(screenWidth, screenHeight, screenScale int) *Context {
	context := &Context{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
		textures:     map[C.GLuint]*Texture{},
	}
	return context
}

func (context *Context) Init() {
	// The main framebuffer should be created sooner than any other
	// framebuffers!
	mainFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &mainFramebuffer)

	context.mainFramebufferTexture = newRenderTargetWithFramebuffer(
		context.screenWidth*context.screenScale,
		context.screenHeight*context.screenScale,
		C.GLuint(mainFramebuffer))

	initializeShaders()

	context.screen = context.NewRenderTarget(
		context.screenWidth, context.screenHeight).(*RenderTarget)
}

func (context *Context) Clear() {
	context.Fill(0, 0, 0)
}

func (context *Context) Fill(r, g, b uint8) {
	const max = float64(math.MaxUint8)
	C.glClearColor(
		C.GLclampf(float64(r)/max),
		C.GLclampf(float64(g)/max),
		C.GLclampf(float64(b)/max),
		1)
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (context *Context) DrawTexture(
	textureID graphics.TextureID,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture := context.textures[C.GLuint(textureID)]

	source := graphics.Rect{0, 0, texture.width, texture.height}
	locations := []graphics.TexturePart{{0, 0, source}}
	context.DrawTextureParts(textureID, locations,
		geometryMatrix, colorMatrix)
}

func (context *Context) DrawTextureParts(
	textureID graphics.TextureID, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture := context.textures[C.GLuint(textureID)]
	if texture == nil {
		panic("invalid texture ID")
	}

	shaderProgram := context.setShaderProgram(geometryMatrix, colorMatrix)
	C.glBindTexture(C.GL_TEXTURE_2D, texture.id)

	vertexAttrLocation := getAttributeLocation(shaderProgram, "vertex")
	textureAttrLocation := getAttributeLocation(shaderProgram, "texture")

	C.glEnableClientState(C.GL_VERTEX_ARRAY)
	C.glEnableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glEnableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glEnableVertexAttribArray(C.GLuint(textureAttrLocation))
	// TODO: Optimization
	for _, part := range parts {
		x1 := float32(part.LocationX)
		x2 := float32(part.LocationX + part.Source.Width)
		y1 := float32(part.LocationY)
		y2 := float32(part.LocationY + part.Source.Height)
		vertex := [...]float32{
			x1, y1,
			x2, y1,
			x1, y2,
			x2, y2,
		}

		src := part.Source
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
		C.glVertexAttribPointer(C.GLuint(vertexAttrLocation), 2,
			C.GL_FLOAT, C.GL_FALSE,
			0, unsafe.Pointer(&vertex[0]))
		C.glVertexAttribPointer(C.GLuint(textureAttrLocation), 2,
			C.GL_FLOAT, C.GL_FALSE,
			0, unsafe.Pointer(&texCoord[0]))
		C.glDrawArrays(C.GL_TRIANGLE_STRIP, 0, 4)
	}
	C.glDisableVertexAttribArray(C.GLuint(textureAttrLocation))
	C.glDisableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glDisableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glDisableClientState(C.GL_VERTEX_ARRAY)
}

func (context *Context) ResetOffscreen() {
	context.setOffscreen(context.screen)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (context *Context) SetOffscreen(renderTargetID graphics.RenderTargetID) {
	renderTarget :=
		(*RenderTarget)(context.textures[C.GLuint(renderTargetID)])
	if renderTarget.framebuffer == 0 {
		renderTarget.framebuffer = createFramebuffer(renderTarget.id)
	}
	context.setOffscreen(renderTarget)
}

func (context *Context) setOffscreen(renderTarget *RenderTarget) {
	context.currentOffscreen = renderTarget

	C.glFlush()

	C.glBindFramebuffer(C.GL_FRAMEBUFFER, renderTarget.framebuffer)
	err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER)
	if err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}

	C.glEnable(C.GL_BLEND)
	C.glBlendFuncSeparate(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA,
		C.GL_ZERO, C.GL_ONE)

	C.glViewport(0, 0, C.GLsizei(abs(renderTarget.textureWidth)),
		C.GLsizei(abs(renderTarget.textureHeight)))
}

func (context *Context) resetOffscreen() {
	context.setOffscreen(context.mainFramebufferTexture)
}

func (context *Context) flush() {
	C.glFlush()
}

func (context *Context) projectionMatrix() [16]float32 {
	texture := context.currentOffscreen

	var e11, e22, e41, e42 float32
	if texture != context.mainFramebufferTexture {
		e11 = float32(2) / float32(texture.textureWidth)
		e22 = float32(2) / float32(texture.textureWidth)
		e41 = -1
		e42 = -1
	} else {
		height := float32(texture.height)
		e11 = float32(2) / float32(texture.textureWidth)
		e22 = -1 * float32(2) / float32(texture.textureHeight)
		e41 = -1
		e42 = -1 + height/float32(texture.textureHeight)*2
	}

	return [...]float32{
		e11, 0, 0, 0,
		0, e22, 0, 0,
		0, 0, 1, 0,
		e41, e42, 0, 1,
	}
}

func (context *Context) setShaderProgram(
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) (program C.GLuint) {
	if colorMatrix.IsIdentity() {
		program = regularShaderProgram
	} else {
		program = colorMatrixShaderProgram
	}
	// TODO: cache and skip?
	C.glUseProgram(program)

	projectionMatrix := context.projectionMatrix()
	C.glUniformMatrix4fv(getUniformLocation(program, "projection_matrix"),
		1, C.GL_FALSE,
		(*C.GLfloat)(&projectionMatrix[0]))

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
		e[0][0], e[1][0], e[2][0], e[3][0],
		e[0][1], e[1][1], e[2][1], e[3][1],
		e[0][2], e[1][2], e[2][2], e[3][2],
		e[0][3], e[1][3], e[2][3], e[3][3],
	}
	C.glUniformMatrix4fv(getUniformLocation(program, "color_matrix"),
		1, C.GL_FALSE, (*C.GLfloat)(&glColorMatrix[0]))

	glColorMatrixTranslation := [...]float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	C.glUniform4fv(getUniformLocation(program, "color_matrix_translation"),
		1, (*C.GLfloat)(&glColorMatrixTranslation[0]))

	return
}

func createFramebuffer(textureID C.GLuint) C.GLuint {
	framebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &framebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, textureID, 0)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) !=
		C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	return framebuffer
}

func (context *Context) NewRenderTarget(width, height int) graphics.RenderTarget {
	renderTarget := newRenderTarget(width, height)
	context.textures[renderTarget.id] = (*Texture)(renderTarget)

	context.SetOffscreen(renderTarget.ID())
	context.Clear()
	context.resetOffscreen()

	return renderTarget
}

func (context *Context) NewTextureFromImage(img image.Image) (
	graphics.Texture, error) {
	texture, err := newTextureFromImage(img)
	if err != nil {
		return nil, err
	}
	context.textures[texture.id] = texture
	return texture, nil
}
