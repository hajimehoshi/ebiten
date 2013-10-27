package texture

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	gtexture "github.com/hajimehoshi/go-ebiten/graphics/texture"
	"image"
	"unsafe"
)

type Native C.GLuint

type Filter int

const (
	FilterLinear = iota
	FilterNearest
)

func createNativeTexture(textureWidth, textureHeight int, pixels []uint8,
	filter Filter) Native {
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
	case FilterLinear:
		glFilter = C.GL_LINEAR
	case FilterNearest:
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

func create(textureWidth, textureHeight int, filter Filter) (interface{}, error) {
	return createNativeTexture(textureWidth, textureHeight, nil, filter), nil
}

func createFromImage(img *image.NRGBA) (interface{}, error) {
	size := img.Bounds().Size()
	return createNativeTexture(size.X, size.Y, img.Pix, FilterLinear), nil
}

func New(width, height int, filter Filter) (*gtexture.Texture, error) {
	f := func(textureWidth, textureHeight int) (interface{}, error) {
		return create(textureWidth, textureHeight, filter)
	}
	return gtexture.New(width, height, f)
}

func NewEmpty(width, height int) (*gtexture.Texture, error) {
	f := func(textureWidth, textureHeight int) (interface{}, error) {
		return nil, nil
	}
	return gtexture.New(width, height, f)
}

func NewFromImage(img image.Image) (*gtexture.Texture, error) {
	return gtexture.NewFromImage(img, createFromImage)
}
