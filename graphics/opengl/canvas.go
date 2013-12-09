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
	"math"
)

type Canvas struct {
	screenId    graphics.RenderTargetId
	ids         *ids
	offscreen   *offscreen.Offscreen
	screenScale int
}

func newCanvas(ids *ids, screenWidth, screenHeight, screenScale int) *Canvas {
	canvas := &Canvas{
		ids:         ids,
		offscreen:   offscreen.New(screenWidth, screenHeight, screenScale),
		screenScale: screenScale,
	}

	var err error
	canvas.screenId, err = ids.CreateRenderTarget(
		screenWidth, screenHeight, texture.FilterNearest)
	if err != nil {
		panic("initializing the offscreen failed: " + err.Error())
	}

	return canvas
}

func (canvas *Canvas) update(draw func(graphics.Canvas)) {
	canvas.init()
	canvas.ResetOffscreen()
	canvas.Clear()

	draw(canvas)

	canvas.flush()
	canvas.setMainFramebufferOffscreen()
	canvas.Clear()

	scale := float64(canvas.screenScale)
	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Scale(scale, scale)
	canvas.DrawRenderTarget(canvas.screenId,
		geometryMatrix, matrix.IdentityColor())
	canvas.flush()
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

// init initializes the canvas. The initial state is saved for each GL canvas.
func (canvas *Canvas) init() {
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
