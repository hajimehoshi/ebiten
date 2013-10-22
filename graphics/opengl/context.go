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
	screenId               graphics.RenderTargetId
	screenWidth            int
	screenHeight           int
	screenScale            int
	textures               map[graphics.TextureId]*Texture
	renderTargets          map[graphics.RenderTargetId]*RenderTarget
	renderTargetToTexture  map[graphics.RenderTargetId]graphics.TextureId
	currentOffscreen       *RenderTarget
	mainFramebufferTexture *RenderTarget
	projectionMatrix       [16]float32
}

func newContext(screenWidth, screenHeight, screenScale int) *Context {
	context := &Context{
		screenWidth:           screenWidth,
		screenHeight:          screenHeight,
		screenScale:           screenScale,
		textures:              map[graphics.TextureId]*Texture{},
		renderTargets:         map[graphics.RenderTargetId]*RenderTarget{},
		renderTargetToTexture: map[graphics.RenderTargetId]graphics.TextureId{},
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

	var err error
	context.screenId, err = context.NewRenderTarget(
		context.screenWidth, context.screenHeight)
	if err != nil {
		panic("initializing the offscreen failed: " + err.Error())
	}
}

func (context *Context) ToTexture(renderTargetId graphics.RenderTargetId) graphics.TextureId {
	return context.renderTargetToTexture[renderTargetId]
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
	textureId graphics.TextureId,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture, ok := context.textures[textureId]
	if !ok {
		panic("invalid texture ID")
	}
	source := graphics.Rect{0, 0, texture.width, texture.height}
	locations := []graphics.TexturePart{{0, 0, source}}
	context.DrawTextureParts(textureId, locations,
		geometryMatrix, colorMatrix)
}

func (context *Context) DrawTextureParts(
	textureId graphics.TextureId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {

	texture, ok := context.textures[textureId]
	if !ok {
		panic("invalid texture ID")
	}

	shaderProgram := context.setShaderProgram(geometryMatrix, colorMatrix)
	C.glBindTexture(C.GL_TEXTURE_2D, texture.Native().(C.GLuint))

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
		tu1 := float32(texture.U(src.X))
		tu2 := float32(texture.U(src.X + src.Width))
		tv1 := float32(texture.V(src.Y))
		tv2 := float32(texture.V(src.Y + src.Height))
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
	context.SetOffscreen(context.screenId)
}

func (context *Context) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	renderTarget := context.renderTargets[renderTargetId]
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

	context.currentOffscreen.SetAsViewport(context)
}

func orthoProjectionMatrix(left, right, bottom, top int) [4][4]float64 {
	e11 := float64(2) / float64(right-left)
	e22 := float64(2) / float64(top-bottom)
	e14 := -1 * float64(right+left) / float64(right-left)
	e24 := -1 * float64(top+bottom) / float64(top-bottom)

	return [4][4]float64{
		{e11, 0, 0, e14},
		{0, e22, 0, e24},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

func (context *Context) SetViewport(x, y, width, height int) {
	C.glViewport(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))

	matrix := orthoProjectionMatrix(x, width, y, height)
	if context.currentOffscreen == context.mainFramebufferTexture {
		// Flip Y and translate
		matrix[1][1] *= -1
		actualHeight := context.screenHeight * context.screenScale
		matrix[1][3] += float64(actualHeight) / float64(height) * 2
	}

	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			context.projectionMatrix[i+j*4] = float32(matrix[i][j])
		}
	}

	// TODO: call 'setShaderProgram' here?
}

func (context *Context) setMainFramebufferOffscreen() {
	context.setOffscreen(context.mainFramebufferTexture)
}

func (context *Context) flush() {
	C.glFlush()
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

func (context *Context) NewRenderTarget(width, height int) (
	graphics.RenderTargetId, error) {
	renderTarget, err := newRenderTarget(width, height)
	if err != nil {
		return 0, nil
	}
	renderTargetId := graphics.RenderTargetId(<-newId)
	textureId := graphics.TextureId(<-newId)
	context.renderTargets[renderTargetId] = renderTarget
	context.textures[textureId] = renderTarget.texture
	context.renderTargetToTexture[renderTargetId] = textureId

	context.setOffscreen(renderTarget)
	context.Clear()
	// TODO: Is it OK to revert he main framebuffer?
	context.setMainFramebufferOffscreen()

	return renderTargetId, nil
}

func (context *Context) NewTextureFromImage(img image.Image) (
	graphics.TextureId, error) {
	texture, err := newTextureFromImage(img)
	if err != nil {
		return 0, err
	}
	textureId := graphics.TextureId(<-newId)
	context.textures[textureId] = texture
	return textureId, nil
}

var newId chan int

func init() {
	newId = make(chan int)
	go func() {
		for i := 0; ; i++ {
			newId <- i
		}
	}()
}
