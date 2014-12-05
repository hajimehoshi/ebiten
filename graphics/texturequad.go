package graphics

import (
	"image"
	"image/draw"
)

type TextureQuad struct {
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

func AdjustImageForTexture(img image.Image) *image.NRGBA {
	width, height := img.Bounds().Size().X, img.Bounds().Size().Y
	adjustedImageBounds := image.Rectangle{
		image.ZP,
		image.Point{
			AdjustSizeForTexture(width),
			AdjustSizeForTexture(height),
		},
	}
	if nrgba, ok := img.(*image.NRGBA); ok &&
		img.Bounds() == adjustedImageBounds {
		return nrgba
	}

	adjustedImage := image.NewNRGBA(adjustedImageBounds)
	dstBounds := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBounds, img, image.ZP, draw.Src)
	return adjustedImage
}

func u(x int, width int) float32 {
	return float32(x) / float32(AdjustSizeForTexture(width))
}

func v(y int, height int) float32 {
	return float32(y) / float32(AdjustSizeForTexture(height))
}

func TextureQuads(parts []TexturePart, width, height int) []TextureQuad {
	quads := []TextureQuad{}
	for _, part := range parts {
		x1 := float32(part.LocationX)
		x2 := float32(part.LocationX + part.Source.Width)
		y1 := float32(part.LocationY)
		y2 := float32(part.LocationY + part.Source.Height)
		u1 := u(part.Source.X, width)
		u2 := u(part.Source.X+part.Source.Width, width)
		v1 := v(part.Source.Y, height)
		v2 := v(part.Source.Y+part.Source.Height, height)
		quad := TextureQuad{x1, x2, y1, y2, u1, u2, v1, v2}
		quads = append(quads, quad)
	}
	return quads
}
