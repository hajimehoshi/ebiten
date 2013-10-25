package texture

import (
	"image"
	"image/draw"
)

func nextPowerOf2(x uint64) uint64 {
	x -= 1
	x |= (x >> 1)
	x |= (x >> 2)
	x |= (x >> 4)
	x |= (x >> 8)
	x |= (x >> 16)
	x |= (x >> 32)
	return x + 1
}

type Texture struct {
	native interface{}
	width  int
	height int
}

func New(width, height int, creator interface {
	Create(textureWidth, textureHeight int) (interface{}, error)
}) (*Texture, error) {
	texture := &Texture{
		width:  width,
		height: height,
	}
	var err error
	texture.native, err = creator.Create(texture.textureWidth(), texture.textureHeight())
	if err != nil {
		return nil, err
	}
	return texture, nil
}

func NewFromImage(img image.Image, creator interface {
	CreateFromImage(img *image.NRGBA) (interface{}, error)
}) (*Texture, error) {
	size := img.Bounds().Size()
	width, height := size.X, size.Y
	texture := &Texture{
		width:  width,
		height: height,
	}
	adjustedImageBound := image.Rectangle{
		image.ZP,
		image.Point{texture.textureWidth(), texture.textureHeight()},
	}
	adjustedImage := image.NewNRGBA(adjustedImageBound)
	dstBound := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBound, img, image.ZP, draw.Src)
	var err error
	texture.native, err = creator.CreateFromImage(adjustedImage)
	if err != nil {
		return nil, err
	}
	return texture, nil
}

func (texture *Texture) Width() int {
	return texture.width
}

func (texture *Texture) Height() int {
	return texture.height
}

func (texture *Texture) textureWidth() int {
	return int(nextPowerOf2(uint64(texture.width)))
}

func (texture *Texture) textureHeight() int {
	return int(nextPowerOf2(uint64(texture.height)))
}

func (texture *Texture) Native() interface{} {
	return texture.native
}

func (texture *Texture) U(x int) float64 {
	return float64(x) / float64(texture.textureWidth())
}

func (texture *Texture) V(y int) float64 {
	return float64(y) / float64(texture.textureHeight())
}

func (texture *Texture) SetAsViewport(x, y int, setter interface{
	SetViewport(x, y, width, height int)
}) {
	setter.SetViewport(x, y, texture.textureWidth(), texture.textureHeight())
}
