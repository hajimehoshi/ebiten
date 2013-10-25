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
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/shader"
	"github.com/hajimehoshi/go-ebiten/graphics/rendertarget"
	"github.com/hajimehoshi/go-ebiten/graphics/texture"
	"image"
	"math"
	"unsafe"
)

type Context struct {
	screenId               graphics.RenderTargetId
	screenWidth            int
	screenHeight           int
	screenScale            int
	textures               map[graphics.TextureId]*texture.Texture
	renderTargets          map[graphics.RenderTargetId]*rendertarget.RenderTarget
	renderTargetToTexture  map[graphics.RenderTargetId]graphics.TextureId
	currentOffscreen       *rendertarget.RenderTarget
	mainFramebufferTexture *rendertarget.RenderTarget
	projectionMatrix       [16]float32
}

func newContext(screenWidth, screenHeight, screenScale int) *Context {
	context := &Context{
		screenWidth:           screenWidth,
		screenHeight:          screenHeight,
		screenScale:           screenScale,
		textures:              map[graphics.TextureId]*texture.Texture{},
		renderTargets:         map[graphics.RenderTargetId]*rendertarget.RenderTarget{},
		renderTargetToTexture: map[graphics.RenderTargetId]graphics.TextureId{},
	}
	return context
}

func (context *Context) Init() {
	// The main framebuffer should be created sooner than any other
	// framebuffers!
	mainFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &mainFramebuffer)

	var err error
	context.mainFramebufferTexture, err = newRenderTargetWithFramebuffer(
		context.screenWidth*context.screenScale,
		context.screenHeight*context.screenScale,
		C.GLuint(mainFramebuffer))
	if err != nil {
		panic("creating main framebuffer failed: " + err.Error())
	}

	shader.Initialize()

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

type TextureDrawing struct {
	context        *Context
	geometryMatrix matrix.Geometry
	colorMatrix    matrix.Color
}

func (t *TextureDrawing) Draw(native interface{}, quads []texture.Quad) {
	if len(quads) == 0 {
		return
	}
	shaderProgram := t.context.setShaderProgram(t.geometryMatrix, t.colorMatrix)
	C.glBindTexture(C.GL_TEXTURE_2D, native.(C.GLuint))

	vertexAttrLocation := shader.GetAttributeLocation(shaderProgram, "vertex")
	textureAttrLocation := shader.GetAttributeLocation(shaderProgram, "texture")

	C.glEnableClientState(C.GL_VERTEX_ARRAY)
	C.glEnableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glEnableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glEnableVertexAttribArray(C.GLuint(textureAttrLocation))
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
	C.glVertexAttribPointer(C.GLuint(textureAttrLocation), 2,
		C.GL_FLOAT, C.GL_FALSE,
		0, unsafe.Pointer(&texCoords[0]))
	C.glDrawElements(C.GL_TRIANGLES, C.GLsizei(len(indicies)),
		C.GL_UNSIGNED_INT, unsafe.Pointer(&indicies[0]))
	C.glDisableVertexAttribArray(C.GLuint(textureAttrLocation))
	C.glDisableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glDisableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glDisableClientState(C.GL_VERTEX_ARRAY)
}

func (context *Context) DrawTexture(
	textureId graphics.TextureId,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture, ok := context.textures[textureId]
	if !ok {
		panic("invalid texture ID")
	}
	drawing := &TextureDrawing{context, geometryMatrix, colorMatrix}
	texture.Draw(drawing.Draw)
}

func (context *Context) DrawTextureParts(
	textureId graphics.TextureId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture, ok := context.textures[textureId]
	if !ok {
		panic("invalid texture ID")
	}
	drawing := &TextureDrawing{context, geometryMatrix, colorMatrix}
	texture.DrawParts(parts, drawing.Draw)
}

func (context *Context) ResetOffscreen() {
	context.SetOffscreen(context.screenId)
}

func (context *Context) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	renderTarget := context.renderTargets[renderTargetId]
	context.setOffscreen(renderTarget)
}

func (context *Context) setOffscreen(renderTarget *rendertarget.RenderTarget) {
	context.currentOffscreen = renderTarget

	C.glFlush()

	framebuffer := renderTarget.Framebuffer().(C.GLuint)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER)
	if err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}

	C.glEnable(C.GL_BLEND)
	C.glBlendFuncSeparate(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA,
		C.GL_ZERO, C.GL_ONE)

	isUsingMainFramebuffer := context.currentOffscreen == context.mainFramebufferTexture
	setter := &viewportSetter{
		isUsingMainFramebuffer,
		context.screenHeight * context.screenScale,
		context,
	}
	context.currentOffscreen.SetAsViewport(setter.Set)
}

type viewportSetter struct {
	isUsingMainFramebuffer bool
	actualScreenHeight     int
	context                *Context
}

func (v *viewportSetter) Set(x, y, width, height int) {
	C.glViewport(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))

	matrix := graphics.OrthoProjectionMatrix(x, width, y, height)
	if v.isUsingMainFramebuffer {
		// Flip Y and move to fit with the top of the window.
		matrix[1][1] *= -1
		matrix[1][3] += float64(v.actualScreenHeight) / float64(height) * 2
	}

	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			v.context.projectionMatrix[i+j*4] = float32(matrix[i][j])
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
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) (program shader.Program) {
	return shader.Use(context.projectionMatrix, geometryMatrix, colorMatrix)
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
	context.textures[textureId] = renderTarget.Texture()
	context.renderTargetToTexture[renderTargetId] = textureId

	context.setOffscreen(renderTarget)
	context.Clear()
	// TODO: Is it OK to revert he main framebuffer?
	context.setMainFramebufferOffscreen()

	return renderTargetId, nil
}

func (context *Context) NewTextureFromImage(img image.Image) (
	graphics.TextureId, error) {
	texture, err := texture.NewFromImage(img, &NativeTextureCreator{})
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
