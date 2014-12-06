package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
	"image"
)

type Texture struct {
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

	ptr := interface{}(nil)
	if pixels != nil {
		ptr = pixels
	}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, textureWidth, textureHeight, 0, gl.RGBA, gl.UNSIGNED_BYTE, ptr)

	return nativeTexture
}

func createTexture(width, height int, filter graphics.Filter) (*Texture, error) {
	w := graphics.AdjustSizeForTexture(width)
	h := graphics.AdjustSizeForTexture(height)
	native := createNativeTexture(w, h,  nil, filter)
	return &Texture{native, width, height}, nil
}

func createTextureFromImage(img image.Image, filter graphics.Filter) (*Texture, error) {
	adjustedImage := graphics.AdjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return &Texture{native, size.X, size.Y}, nil
}

func (t *Texture) dispose() {
	t.native.Delete()
}
