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

func createNativeTexture(textureWidth, textureHeight int, pixels []uint8) Native {
	nativeTexture := C.GLuint(0)

	C.glGenTextures(1, (*C.GLuint)(&nativeTexture))
	if nativeTexture < 0 {
		panic("glGenTexture failed")
	}
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(nativeTexture))
	defer C.glBindTexture(C.GL_TEXTURE_2D, 0)

	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)

	ptr := unsafe.Pointer(nil)
	if pixels != nil {
		ptr = unsafe.Pointer(&pixels[0])
	}
	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_RGBA,
		C.GLsizei(textureWidth), C.GLsizei(textureHeight),
		0, C.GL_RGBA, C.GL_UNSIGNED_BYTE, ptr)

	return Native(nativeTexture)
}

func create(textureWidth, textureHeight int) (interface{}, error) {
	return createNativeTexture(textureWidth, textureHeight, nil), nil
}

func createFromImage(img *image.NRGBA) (interface{}, error) {
	size := img.Bounds().Size()
	return createNativeTexture(size.X, size.Y, img.Pix), nil
}

func New(width, height int) (*gtexture.Texture, error) {
	return gtexture.New(width, height, create)
}

func NewFromImage(img image.Image) (*gtexture.Texture, error) {
	return gtexture.NewFromImage(img, createFromImage)
}
