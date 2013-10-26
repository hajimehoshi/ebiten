package texture

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
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

func New(width, height int, create func(textureWidth, textureHeight int) (interface{}, error)) (*Texture, error) {
	texture := &Texture{
		width:  width,
		height: height,
	}
	var err error
	texture.native, err = create(texture.textureWidth(), texture.textureHeight())
	if err != nil {
		return nil, err
	}
	return texture, nil
}

func NewFromImage(img image.Image, create func(img *image.NRGBA) (interface{}, error)) (*Texture, error) {
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
	texture.native, err = create(adjustedImage)
	if err != nil {
		return nil, err
	}
	return texture, nil
}

func (texture *Texture) textureWidth() int {
	return int(nextPowerOf2(uint64(texture.width)))
}

func (texture *Texture) textureHeight() int {
	return int(nextPowerOf2(uint64(texture.height)))
}

// TODO: Remove this
func (texture *Texture) Native() interface{} {
	return texture.native
}

func (texture *Texture) u(x int) float32 {
	return float32(x) / float32(texture.textureWidth())
}

func (texture *Texture) v(y int) float32 {
	return float32(y) / float32(texture.textureHeight())
}

func (texture *Texture) SetAsViewport(setter func(x, y, width, height int)) {
	setter(0, 0, texture.textureWidth(), texture.textureHeight())
}

type Quad struct {
	VertexX1       float32
	VertexX2       float32
	VertexY1       float32
	VertexY2       float32
	TextureCoordU1 float32
	TextureCoordU2 float32
	TextureCoordV1 float32
	TextureCoordV2 float32
}

func (texture *Texture) Draw(draw func(native interface{}, quads []Quad)) {
	x1 := float32(0)
	x2 := float32(texture.width)
	y1 := float32(0)
	y2 := float32(texture.height)
	u1 := texture.u(0)
	u2 := texture.u(texture.width)
	v1 := texture.v(0)
	v2 := texture.v(texture.height)
	quad := Quad{x1, x2, y1, y2, u1, u2, v1, v2}
	draw(texture.native, []Quad{quad})
}

func (texture *Texture) DrawParts(parts []graphics.TexturePart, draw func(native interface{}, quads []Quad)) {
	quads := []Quad{}
	for _, part := range parts {
		x1 := float32(part.LocationX)
		x2 := float32(part.LocationX + part.Source.Width)
		y1 := float32(part.LocationY)
		y2 := float32(part.LocationY + part.Source.Height)
		u1 := texture.u(part.Source.X)
		u2 := texture.u(part.Source.X + part.Source.Width)
		v1 := texture.v(part.Source.Y)
		v2 := texture.v(part.Source.Y + part.Source.Height)
		quad := Quad{x1, x2, y1, y2, u1, u2, v1, v2}
		quads = append(quads, quad)
	}
	draw(texture.native, quads)
}
