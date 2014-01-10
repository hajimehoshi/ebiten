package offscreen

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Texture interface {
	Draw(projectionMatrix [4][4]float64,
		geometryMatrix matrix.Geometry, colorMatrix matrix.Color)
	DrawParts(parts []graphics.TexturePart, projectionMatrix [4][4]float64,
		geometryMatrix matrix.Geometry, colorMatrix matrix.Color)
}

type RenderTarget interface {
	SetAsViewport()
	ProjectionMatrix() [4][4]float64
}

type Offscreen struct {
	currentRenderTarget RenderTarget
	mainFramebuffer     RenderTarget
}

func New(mainFramebuffer RenderTarget) *Offscreen {
	return &Offscreen{
		mainFramebuffer: mainFramebuffer,
	}
}

func (o *Offscreen) Set(rt RenderTarget) {
	o.currentRenderTarget = rt
	rt.SetAsViewport()
}

func (o *Offscreen) SetMainFramebuffer() {
	o.Set(o.mainFramebuffer)
}

func (o *Offscreen) DrawTexture(texture Texture,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	projectionMatrix := o.currentRenderTarget.ProjectionMatrix()
	texture.Draw(projectionMatrix, geometryMatrix, colorMatrix)
}

func (o *Offscreen) DrawTextureParts(texture Texture,
	parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	projectionMatrix := o.currentRenderTarget.ProjectionMatrix()
	texture.DrawParts(parts, projectionMatrix, geometryMatrix, colorMatrix)
}
