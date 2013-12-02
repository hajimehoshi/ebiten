package graphics

import (
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Canvas interface {
	Clear()
	Fill(r, g, b uint8)
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
}

type LazyCanvas struct {
	funcs []func(Canvas)
}

func NewLazyCanvas() *LazyCanvas {
	return &LazyCanvas{
		funcs: []func(Canvas){},
	}
}

func (c *LazyCanvas) Flush(actual Canvas) {
	for _, f := range c.funcs {
		f(actual)
	}
	c.funcs = []func(Canvas){}
}

func (c *LazyCanvas) Clear() {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.Clear()
	})
}

func (c *LazyCanvas) Fill(r, g, b uint8) {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.Fill(r, g, b)
	})
}

func (c *LazyCanvas) DrawTexture(id TextureId,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.DrawTexture(id, geometryMatrix, colorMatrix)
	})
}

func (c *LazyCanvas) DrawRenderTarget(id RenderTargetId,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.DrawRenderTarget(id, geometryMatrix, colorMatrix)
	})
}

func (c *LazyCanvas) DrawTextureParts(id TextureId,
	parts []TexturePart,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.DrawTextureParts(id, parts, geometryMatrix, colorMatrix)
	})
}

func (c *LazyCanvas) DrawRenderTargetParts(id RenderTargetId,
	parts []TexturePart,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.DrawRenderTargetParts(id, parts, geometryMatrix, colorMatrix)
	})
}

func (c *LazyCanvas) ResetOffscreen() {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.ResetOffscreen()
	})
}

func (c *LazyCanvas) SetOffscreen(id RenderTargetId) {
	c.funcs = append(c.funcs, func(actual Canvas) {
		actual.SetOffscreen(id)
	})
}
