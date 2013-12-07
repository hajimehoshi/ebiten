package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/offscreen"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
	"image"
	"math"
)

type Context struct {
	screenId  graphics.RenderTargetId
	ids       *ids
	offscreen *offscreen.Offscreen
}

func newContext(screenWidth, screenHeight, screenScale int) *Context {
	context := &Context{
		ids:       newIds(),
		offscreen: offscreen.New(screenWidth, screenHeight, screenScale),
	}

	var err error
	context.screenId, err = context.createRenderTarget(
		screenWidth, screenHeight, texture.FilterNearest)
	if err != nil {
		panic("initializing the offscreen failed: " + err.Error())
	}

	context.Init()

	return context
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
	id graphics.TextureId,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	tex := context.ids.TextureAt(id)
	context.offscreen.DrawTexture(tex, geometryMatrix, colorMatrix)
}

func (context *Context) DrawRenderTarget(
	id graphics.RenderTargetId,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	context.DrawTexture(context.ids.ToTexture(id), geometryMatrix, colorMatrix)
}

func (context *Context) DrawTextureParts(
	id graphics.TextureId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	tex := context.ids.TextureAt(id)
	context.offscreen.DrawTextureParts(tex, parts, geometryMatrix, colorMatrix)
}

func (context *Context) DrawRenderTargetParts(
	id graphics.RenderTargetId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	context.DrawTextureParts(context.ids.ToTexture(id), parts, geometryMatrix, colorMatrix)
}

// Init initializes the context. The initial state is saved for each GL context.
func (context *Context) Init() {
	C.glEnable(C.GL_TEXTURE_2D)
	C.glEnable(C.GL_BLEND)
}

func (context *Context) ResetOffscreen() {
	context.SetOffscreen(context.screenId)
}

func (context *Context) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	renderTarget := context.ids.RenderTargetAt(renderTargetId)
	context.offscreen.Set(renderTarget)
}

func (context *Context) setMainFramebufferOffscreen() {
	context.offscreen.SetMainFramebuffer()
}

func (context *Context) flush() {
	C.glFlush()
}

func (context *Context) createRenderTarget(width, height int, filter texture.Filter) (
	graphics.RenderTargetId, error) {
	renderTargetId, err := context.ids.CreateRenderTarget(width, height, filter)
	if err != nil {
		return 0, err
	}
	return renderTargetId, nil
}

func (context *Context) CreateRenderTarget(width, height int) (
	graphics.RenderTargetId, error) {
	return context.createRenderTarget(width, height, texture.FilterLinear)
}

func (context *Context) CreateTextureFromImage(img image.Image) (
	graphics.TextureId, error) {
	return context.ids.CreateTextureFromImage(img)
}
