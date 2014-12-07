package shader

import (
	"github.com/hajimehoshi/ebiten/graphics"
)

type textureQuad struct {
	VertexX1       float32
	VertexX2       float32
	VertexY1       float32
	VertexY2       float32
	TextureCoordU1 float32
	TextureCoordU2 float32
	TextureCoordV1 float32
	TextureCoordV2 float32
}

func NextPowerOf2(x uint64) uint64 {
	x -= 1
	x |= (x >> 1)
	x |= (x >> 2)
	x |= (x >> 4)
	x |= (x >> 8)
	x |= (x >> 16)
	x |= (x >> 32)
	return x + 1
}

func AdjustSizeForTexture(size int) int {
	return int(NextPowerOf2(uint64(size)))
}

func u(x int, width int) float32 {
	return float32(x) / float32(AdjustSizeForTexture(width))
}

func v(y int, height int) float32 {
	return float32(y) / float32(AdjustSizeForTexture(height))
}

func textureQuads(parts []graphics.TexturePart, width, height int) []textureQuad {
	quads := []textureQuad{}
	for _, part := range parts {
		x1 := float32(part.LocationX)
		x2 := float32(part.LocationX + part.Source.Width)
		y1 := float32(part.LocationY)
		y2 := float32(part.LocationY + part.Source.Height)
		u1 := u(part.Source.X, width)
		u2 := u(part.Source.X+part.Source.Width, width)
		v1 := v(part.Source.Y, height)
		v2 := v(part.Source.Y+part.Source.Height, height)
		quad := textureQuad{x1, x2, y1, y2, u1, u2, v1, v2}
		quads = append(quads, quad)
	}
	return quads
}
