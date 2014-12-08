package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/opengl/internal/shader"
	"image"
	"image/draw"
)

func adjustImageForTexture(img image.Image) *image.NRGBA {
	width, height := img.Bounds().Size().X, img.Bounds().Size().Y
	adjustedImageBounds := image.Rectangle{
		image.ZP,
		image.Point{
			shader.AdjustSizeForTexture(width),
			shader.AdjustSizeForTexture(height),
		},
	}
	if nrgba, ok := img.(*image.NRGBA); ok && img.Bounds() == adjustedImageBounds {
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

type texture struct {
	native gl.Texture
	width  int
	height int
}

func createNativeTexture(textureWidth, textureHeight int, pixels []uint8, filter graphics.Filter) gl.Texture {
	nativeTexture := gl.GenTexture()
	if nativeTexture < 0 {
		panic("glGenTexture failed")
	}
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	nativeTexture.Bind(gl.TEXTURE_2D)
	defer gl.Texture(0).Bind(gl.TEXTURE_2D)

	glFilter := 0
	switch filter {
	case graphics.FilterLinear:
		glFilter = gl.LINEAR
	case graphics.FilterNearest:
		glFilter = gl.NEAREST
	default:
		panic("not reached")
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, glFilter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, glFilter)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, textureWidth, textureHeight, 0, gl.RGBA, gl.UNSIGNED_BYTE, pixels)

	return nativeTexture
}

func createTexture(width, height int, filter graphics.Filter) (*texture, error) {
	w := shader.AdjustSizeForTexture(width)
	h := shader.AdjustSizeForTexture(height)
	native := createNativeTexture(w, h, nil, filter)
	return &texture{native, width, height}, nil
}

func createTextureFromImage(img image.Image, filter graphics.Filter) (*texture, error) {
	adjustedImage := adjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return &texture{native, size.X, size.Y}, nil
}

func (t *texture) dispose() {
	t.native.Delete()
}
