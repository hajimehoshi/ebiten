package graphics

import (
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Context interface {
	Clear()
	Fill(r, g, b uint8)
	// TODO: Refacotring
	DrawTexture(id TextureId,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawRenderTarget(id RenderTargetId,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(id TextureId,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawRenderTargetParts(id RenderTargetId,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)

	ResetOffscreen()
	SetOffscreen(id RenderTargetId)

	// TODO: glTextureSubImage2D
}

type LazyContext struct {
	funcs []func(Context)
}

func NewLazyContext() *LazyContext {
	return &LazyContext{
		funcs: []func(Context){},
	}
}

func (c *LazyContext) Flush(actual Context) {
	for _, f := range c.funcs {
		f(actual)
	}
	c.funcs = []func(Context){}
}

func (c *LazyContext) Clear() {
	c.funcs = append(c.funcs, func(actual Context) {
		actual.Clear()
	})
}

func (c *LazyContext) Fill(r, g, b uint8) {
	c.funcs = append(c.funcs, func(actual Context) {
		actual.Fill(r, g, b)
	})
}

func (c *LazyContext) DrawTexture(id TextureId,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	c.funcs = append(c.funcs, func(actual Context) {
		actual.DrawTexture(id, geometryMatrix, colorMatrix)
	})
}

func (c *LazyContext) DrawRenderTarget(id RenderTargetId,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	c.funcs = append(c.funcs, func(actual Context) {
		actual.DrawRenderTarget(id, geometryMatrix, colorMatrix)
	})
}

func (c *LazyContext) DrawTextureParts(id TextureId,
	parts []TexturePart,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	parts2 := make([]TexturePart, len(parts))
	copy(parts2, parts)
	c.funcs = append(c.funcs, func(actual Context) {
		actual.DrawTextureParts(id, parts2, geometryMatrix, colorMatrix)
	})
}

func (c *LazyContext) DrawRenderTargetParts(id RenderTargetId,
	parts []TexturePart,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	parts2 := make([]TexturePart, len(parts))
	copy(parts2, parts)
	c.funcs = append(c.funcs, func(actual Context) {
		actual.DrawRenderTargetParts(id, parts2, geometryMatrix, colorMatrix)
	})
}

func (c *LazyContext) ResetOffscreen() {
	c.funcs = append(c.funcs, func(actual Context) {
		actual.ResetOffscreen()
	})
}

func (c *LazyContext) SetOffscreen(id RenderTargetId) {
	c.funcs = append(c.funcs, func(actual Context) {
		actual.SetOffscreen(id)
	})
}
