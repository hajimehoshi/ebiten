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

type Canvas struct {
	screenId  graphics.RenderTargetId
	ids       *ids
	offscreen *offscreen.Offscreen
}

func newCanvas(screenWidth, screenHeight, screenScale int) *Canvas {
	canvas := &Canvas{
		ids:       newIds(),
		offscreen: offscreen.New(screenWidth, screenHeight, screenScale),
	}

	var err error
	canvas.screenId, err = canvas.createRenderTarget(
		screenWidth, screenHeight, texture.FilterNearest)
	if err != nil {
		panic("initializing the offscreen failed: " + err.Error())
	}

	canvas.Init()

	return canvas
}

func (canvas *Canvas) Clear() {
	canvas.Fill(0, 0, 0)
}

func (canvas *Canvas) Fill(r, g, b uint8) {
	const max = float64(math.MaxUint8)
	C.glClearColor(
		C.GLclampf(float64(r)/max),
		C.GLclampf(float64(g)/max),
		C.GLclampf(float64(b)/max),
		1)
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (canvas *Canvas) DrawTexture(
	id graphics.TextureId,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	tex := canvas.ids.TextureAt(id)
	canvas.offscreen.DrawTexture(tex, geometryMatrix, colorMatrix)
}

func (canvas *Canvas) DrawRenderTarget(
	id graphics.RenderTargetId,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	canvas.DrawTexture(canvas.ids.ToTexture(id), geometryMatrix, colorMatrix)
}

func (canvas *Canvas) DrawTextureParts(
	id graphics.TextureId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	tex := canvas.ids.TextureAt(id)
	canvas.offscreen.DrawTextureParts(tex, parts, geometryMatrix, colorMatrix)
}

func (canvas *Canvas) DrawRenderTargetParts(
	id graphics.RenderTargetId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	canvas.DrawTextureParts(canvas.ids.ToTexture(id), parts, geometryMatrix, colorMatrix)
}

// Init initializes the canvas. The initial state is saved for each GL canvas.
func (canvas *Canvas) Init() {
	C.glEnable(C.GL_TEXTURE_2D)
	C.glEnable(C.GL_BLEND)
}

func (canvas *Canvas) ResetOffscreen() {
	canvas.SetOffscreen(canvas.screenId)
}

func (canvas *Canvas) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	renderTarget := canvas.ids.RenderTargetAt(renderTargetId)
	canvas.offscreen.Set(renderTarget)
}

func (canvas *Canvas) setMainFramebufferOffscreen() {
	canvas.offscreen.SetMainFramebuffer()
}

func (canvas *Canvas) flush() {
	C.glFlush()
}

func (canvas *Canvas) createRenderTarget(width, height int, filter texture.Filter) (
	graphics.RenderTargetId, error) {
	renderTargetId, err := canvas.ids.CreateRenderTarget(width, height, filter)
	if err != nil {
		return 0, err
	}
	return renderTargetId, nil
}

func (canvas *Canvas) CreateRenderTarget(width, height int) (
	graphics.RenderTargetId, error) {
	return canvas.createRenderTarget(width, height, texture.FilterLinear)
}

func (canvas *Canvas) CreateTextureFromImage(img image.Image) (
	graphics.TextureId, error) {
	return canvas.ids.CreateTextureFromImage(img)
}
