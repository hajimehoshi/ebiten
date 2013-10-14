package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"image"
	"unsafe"
)

func clp2(x uint64) uint64 {
	x -= 1
	x |= (x >> 1)
	x |= (x >> 2)
	x |= (x >> 4)
	x |= (x >> 8)
	x |= (x >> 16)
	x |= (x >> 32)
	return x + 1
}

func adjustPixels(width, height int, pixels []uint8) []uint8 {
	textureWidth := int(clp2(uint64(width)))
	textureHeight := int(clp2(uint64(height)))
	if width == textureWidth && height == textureHeight {
		return pixels
	}

	newPixels := make([]uint8, textureWidth*textureHeight*4)

	for j := 0; j < height; j++ {
		copy(newPixels[textureWidth*4*j:],
			pixels[width*4*j:width*4*j+width*4])
	}
	return newPixels
}

type Texture struct {
	id            C.GLuint
	width         int
	height        int
	textureWidth  int
	textureHeight int
	framebuffer   C.GLuint
}

func (texture *Texture) ID() graphics.TextureID {
	return graphics.TextureID(texture.id)
}

func (texture *Texture) Width() int {
	return texture.width
}

func (texture *Texture) Height() int {
	return texture.height
}

func createTexture(width, height int, pixels []uint8) *Texture {
	if pixels != nil {
		pixels = adjustPixels(width, height, pixels)
	}
	textureWidth := int(clp2(uint64(width)))
	textureHeight := int(clp2(uint64(height)))
	texture := &Texture{
		id:            0,
		width:         width,
		height:        height,
		textureWidth:  textureWidth,
		textureHeight: textureHeight,
	}

	textureID := C.GLuint(0)
	C.glGenTextures(1, (*C.GLuint)(&textureID))
	if textureID < 0 {
		panic("glGenTexture failed")
	}
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(textureID))

	ptr := unsafe.Pointer(nil)
	if pixels != nil {
		ptr = unsafe.Pointer(&pixels[0])
	}
	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_RGBA,
		C.GLsizei(textureWidth), C.GLsizei(textureHeight),
		0, C.GL_RGBA, C.GL_UNSIGNED_BYTE, ptr)

	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glBindTexture(C.GL_TEXTURE_2D, 0)

	texture.id = textureID

	return texture
}

type textureError string

func (err textureError) Error() string {
	return "Texture Error: " + string(err)
}

func newRenderTarget(width, height int) *RenderTarget {
	renderTarget := createTexture(width, height, nil)
	return (*RenderTarget)(renderTarget)
}

func newTextureFromImage(img image.Image) (*Texture, error) {
	var pix []uint8
	switch img.(type) {
	case *image.RGBA:
		pix = img.(*image.RGBA).Pix
	case *image.NRGBA:
		pix = img.(*image.NRGBA).Pix
	default:
		return nil, textureError("image format must be RGBA or NRGBA")
	}
	size := img.Bounds().Size()
	return createTexture(size.X, size.Y, pix), nil
}

func newRenderTargetWithFramebuffer(width, height int,
	framebuffer C.GLuint) *RenderTarget {
	texture := &Texture{
		id:            0,
		width:         width,
		height:        height,
		textureWidth:  int(clp2(uint64(width))),
		textureHeight: int(clp2(uint64(height))),
		framebuffer:   framebuffer,
	}
	return (*RenderTarget)(texture)
}

type RenderTarget Texture

func (renderTarget *RenderTarget) Texture() graphics.Texture {
	return (*Texture)(renderTarget)
}

func (renderTarget *RenderTarget) ID() graphics.RenderTargetID {
	return graphics.RenderTargetID(renderTarget.id)
}

func (renderTarget *RenderTarget) Width() int {
	return renderTarget.width
}

func (renderTarget *RenderTarget) Height() int {
	return renderTarget.height
}
