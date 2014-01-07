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
