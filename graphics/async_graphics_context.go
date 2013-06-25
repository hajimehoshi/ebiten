package graphics

import (
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image/color"
)

type AsyncGraphicsContext struct {
	ch chan<- func(GraphicsContext)
}

func NewAsyncGraphicsContext(ch chan<- func(GraphicsContext)) *AsyncGraphicsContext {
	return &AsyncGraphicsContext{
		ch: ch,
	}
}

func (g *AsyncGraphicsContext) Clear() {
	g.ch <- func(g2 GraphicsContext) {
		g2.Clear()
	}
}

func (g *AsyncGraphicsContext) Fill(clr color.Color) {
	g.ch <- func(g2 GraphicsContext) {
		g2.Fill(clr)
	}
}

func (g *AsyncGraphicsContext) DrawRect(rect Rect, clr color.Color) {
	g.ch <- func(g2 GraphicsContext) {
		g2.DrawRect(rect, clr)
	}
}

func (g *AsyncGraphicsContext) DrawTexture(textureID TextureID,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	g.ch <- func(g2 GraphicsContext) {
		g2.DrawTexture(textureID, geometryMatrix, colorMatrix)
	}
}

func (g *AsyncGraphicsContext) DrawTextureParts(textureID TextureID,
	locations []TexturePart,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	g.ch <- func(g2 GraphicsContext) {
		g2.DrawTextureParts(textureID, locations, geometryMatrix, colorMatrix)
	}
}

func (g *AsyncGraphicsContext) SetOffscreen(textureID TextureID) {
	g.ch <- func(g2 GraphicsContext) {
		g2.SetOffscreen(textureID)
	}
}
