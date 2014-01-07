package texture

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	gtexture "github.com/hajimehoshi/go-ebiten/graphics/texture"
	"image"
	"unsafe"
)

type Native C.GLuint

func createNativeTexture(textureWidth, textureHeight int, pixels []uint8,
	filter graphics.Filter) Native {
	nativeTexture := C.GLuint(0)

	C.glGenTextures(1, (*C.GLuint)(&nativeTexture))
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

	return Native(nativeTexture)
}

func Create(width, height int, filter graphics.Filter) (*gtexture.Texture, error) {
	native := createNativeTexture(
		graphics.AdjustSizeForTexture(width),
		graphics.AdjustSizeForTexture(height), nil, filter)
	return gtexture.New(native, width, height), nil
}

func CreateFromImage(img image.Image, filter graphics.Filter) (*gtexture.Texture, error) {
	adjustedImage := graphics.AdjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return gtexture.New(native, size.X, size.Y), nil
}
