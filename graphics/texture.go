package graphics

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"image"
	"unsafe"
	"../ui"
)

func Clp2(x uint64) uint64 {
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
	id C.GLuint
	Width int
	Height int
	TextureWidth int
	TextureHeight int
}

func createTexture(width, height int, pixels []uint8) *Texture{
	textureWidth := int(Clp2(uint64(width)))
	textureHeight := int(Clp2(uint64(height)))
	if width != textureWidth {
		panic("sorry, but width should be power of 2")
	}
	if height != textureHeight {
		panic("sorry, but height should be power of 2")
	}
	texture := &Texture{
		id: 0,
		Width: width,
		Height: height,
		TextureWidth: textureWidth,
		TextureHeight: textureHeight,
	}

	ch := make(chan C.GLuint)
	ui.ExecuteOnUIThread(func() {
		textureID := C.GLuint(0)
		C.glGenTextures(1, (*C.GLuint)(&textureID))
		C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
		C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(textureID))
		if textureID != 0 {
			panic("glBindTexture failed")
		}
		
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

		ch<- textureID
		close(ch)
	})
	// TODO: should wait?
	go func() {
		texture.id = <-ch
	}()

	return texture
}

func NewTexture(width, height int) *Texture {
	return createTexture(width, height, nil)
}

func NewTextureFromRGBA(image *image.RGBA) *Texture {
	return createTexture(image.Rect.Size().X, image.Rect.Size().Y, image.Pix)
}

func (texture *Texture) IsAvailable() bool {
	return texture.id != 0
}

func init() {
	// TODO: Initialize OpenGL here?
}
