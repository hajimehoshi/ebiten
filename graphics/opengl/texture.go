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

type Texture struct {
	native C.GLuint
	width  int
	height int
}

func createNativeTexture(
	textureWidth, textureHeight int,
	pixels []uint8,
	filter graphics.Filter) C.GLuint {
	nativeTexture := C.GLuint(0)

	C.glGenTextures(1, &nativeTexture)
	if nativeTexture < 0 {
		panic("glGenTexture failed")
	}
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(nativeTexture))
	defer C.glBindTexture(C.GL_TEXTURE_2D, 0)

	glFilter := C.GLint(0)
	switch filter {
	case graphics.FilterLinear:
		glFilter = C.GL_LINEAR
	case graphics.FilterNearest:
		glFilter = C.GL_NEAREST
	default:
		panic("not reached")
	}
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, glFilter)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, glFilter)

	ptr := unsafe.Pointer(nil)
	if pixels != nil {
		ptr = unsafe.Pointer(&pixels[0])
	}
	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_RGBA,
		C.GLsizei(textureWidth), C.GLsizei(textureHeight),
		0, C.GL_RGBA, C.GL_UNSIGNED_BYTE, ptr)

	return nativeTexture
}

func createTexture(
	width, height int,
	filter graphics.Filter) (*Texture, error) {
	native := createNativeTexture(
		graphics.AdjustSizeForTexture(width),
		graphics.AdjustSizeForTexture(height),
		nil,
		filter)
	return &Texture{native, width, height}, nil
}

func createTextureFromImage(
	img image.Image,
	filter graphics.Filter) (*Texture, error) {
	adjustedImage := graphics.AdjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return &Texture{native, size.X, size.Y}, nil
}

func (t *Texture) dispose() {
	C.glDeleteTextures(1, &t.native)
}
