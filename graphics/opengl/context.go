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
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/shader"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
	grendertarget "github.com/hajimehoshi/go-ebiten/graphics/rendertarget"
	gtexture "github.com/hajimehoshi/go-ebiten/graphics/texture"
	"image"
	"math"
)

type Context struct {
	screenId               graphics.RenderTargetId
	screenWidth            int
	screenHeight           int
	screenScale            int
	textures               map[graphics.TextureId]*gtexture.Texture
	renderTargets          map[graphics.RenderTargetId]*grendertarget.RenderTarget
	renderTargetToTexture  map[graphics.RenderTargetId]graphics.TextureId
	mainFramebufferTexture *grendertarget.RenderTarget
	projectionMatrix       [16]float32
}

func newContext(screenWidth, screenHeight, screenScale int) *Context {
	context := &Context{
		screenWidth:           screenWidth,
		screenHeight:          screenHeight,
		screenScale:           screenScale,
		textures:              map[graphics.TextureId]*gtexture.Texture{},
		renderTargets:         map[graphics.RenderTargetId]*grendertarget.RenderTarget{},
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
	context.mainFramebufferTexture, err = rendertarget.NewWithFramebuffer(
		context.screenWidth*context.screenScale,
		context.screenHeight*context.screenScale,
		rendertarget.Framebuffer(mainFramebuffer),
		texture.FilterLinear)
	if err != nil {
		panic("creating main framebuffer failed: " + err.Error())
	}

	context.screenId, err = context.newRenderTarget(
		context.screenWidth, context.screenHeight, texture.FilterNearest)
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
	tex, ok := context.textures[textureId]
	if !ok {
		panic("invalid texture ID")
	}
	tex.Draw(func(native interface{}, quads []gtexture.Quad) {
		shader.DrawTexture(native.(texture.Native),
			context.projectionMatrix, quads,
			geometryMatrix, colorMatrix)
	})
}

func (context *Context) DrawTextureParts(
	textureId graphics.TextureId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	tex, ok := context.textures[textureId]
	if !ok {
		panic("invalid texture ID")
	}
	tex.DrawParts(parts, func(native interface{}, quads []gtexture.Quad) {
		shader.DrawTexture(native.(texture.Native),
			context.projectionMatrix, quads,
			geometryMatrix, colorMatrix)
	})
}

func (context *Context) ResetOffscreen() {
	context.SetOffscreen(context.screenId)
}

func (context *Context) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	renderTarget := context.renderTargets[renderTargetId]
	context.setOffscreen(renderTarget)
}

func (context *Context) setOffscreen(renderTarget *grendertarget.RenderTarget) {
	C.glFlush()

	renderTarget.SetAsOffscreen(func(framebuffer interface{}) {
		f := framebuffer.(rendertarget.Framebuffer)
		C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(f))
		err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER)
		if err != C.GL_FRAMEBUFFER_COMPLETE {
			panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
		}

		C.glEnable(C.GL_BLEND)
		C.glBlendFuncSeparate(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA,
			C.GL_ZERO, C.GL_ONE)

		isUsingMainFramebuffer := renderTarget == context.mainFramebufferTexture
		setter := &viewportSetter{
			isUsingMainFramebuffer,
			context.screenHeight * context.screenScale,
			context,
		}
		renderTarget.SetAsViewport(setter.Set)
	})
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
}

func (context *Context) setMainFramebufferOffscreen() {
	context.setOffscreen(context.mainFramebufferTexture)
}

func (context *Context) flush() {
	C.glFlush()
}

func (context *Context) newRenderTarget(width, height int, filter texture.Filter) (
	graphics.RenderTargetId, error) {
	renderTarget, err := rendertarget.New(width, height, filter)
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

func (context *Context) NewRenderTarget(width, height int) (
	graphics.RenderTargetId, error) {
	return context.newRenderTarget(width, height, texture.FilterLinear)
}

func (context *Context) NewTextureFromImage(img image.Image) (
	graphics.TextureId, error) {
	texture, err := texture.NewFromImage(img)
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
