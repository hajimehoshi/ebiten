package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
)

type Context struct {
	screenId    graphics.RenderTargetId
	mainId      graphics.RenderTargetId
	currentId   graphics.RenderTargetId
	ids         *ids
	screenScale int
}

func newContext(ids *ids, screenWidth, screenHeight, screenScale int) *Context {
	mainFramebuffer := rendertarget.NewWithCurrentFramebuffer(
		screenWidth*screenScale,
		screenHeight*screenScale)
	context := &Context{
		ids:         ids,
		screenScale: screenScale,
	}
	context.mainId = context.ids.AddRenderTarget(mainFramebuffer)

	var err error
	context.screenId, err = ids.CreateRenderTarget(
		screenWidth, screenHeight, graphics.FilterNearest)
	if err != nil {
		panic("initializing the offscreen failed: " + err.Error())
	}
	context.ResetOffscreen()
	context.Clear()

	return context
}

func (context *Context) Dispose() {
	// TODO: remove main framebuffer?
	context.ids.DeleteRenderTarget(context.screenId)
}

func (context *Context) update(draw func(graphics.Context)) {
	C.glEnable(C.GL_TEXTURE_2D)
	C.glEnable(C.GL_BLEND)

	context.ResetOffscreen()
	context.Clear()

	draw(context)

	C.glFlush()
	context.SetOffscreen(context.mainId)
	context.Clear()

	scale := float64(context.screenScale)
	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Scale(scale, scale)
	context.DrawRenderTarget(context.screenId,
		geometryMatrix, matrix.IdentityColor())
	C.glFlush()
}

func (context *Context) Clear() {
	context.Fill(0, 0, 0)
}

func (context *Context) Fill(r, g, b uint8) {
	context.ids.FillRenderTarget(context.currentId, r, g, b)
}

func (context *Context) DrawTexture(
	id graphics.TextureId, geo matrix.Geometry, color matrix.Color) {
	context.ids.DrawTexture(context.currentId, id, geo, color)
}

func (context *Context) DrawRenderTarget(
	id graphics.RenderTargetId,
	geo matrix.Geometry, color matrix.Color) {
	context.ids.DrawRenderTarget(context.currentId, id, geo, color)
}

func (context *Context) DrawTextureParts(
	id graphics.TextureId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	context.ids.DrawTextureParts(context.currentId, id, parts, geometryMatrix, colorMatrix)
}

func (context *Context) DrawRenderTargetParts(
	id graphics.RenderTargetId, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	context.ids.DrawRenderTargetParts(context.currentId, id, parts, geometryMatrix, colorMatrix)
}

func (context *Context) ResetOffscreen() {
	context.currentId = context.screenId
}

func (context *Context) SetOffscreen(renderTargetId graphics.RenderTargetId) {
	context.currentId = renderTargetId
}
