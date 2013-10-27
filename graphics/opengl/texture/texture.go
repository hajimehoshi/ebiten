package texture

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics/rendertarget"
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

type Framebuffer C.GLuint

func NewRenderTarget(width, height int) (*rendertarget.RenderTarget, error) {
	texture, err := gtexture.New(width, height, create)
	if err != nil {
		return nil, err
	}
	framebuffer := createFramebuffer(C.GLuint(texture.Native().(Native)))
	return rendertarget.NewWithFramebuffer(texture, Framebuffer(framebuffer)), nil
}

func NewRenderTargetWithFramebuffer(width, height int, framebuffer Framebuffer) (*rendertarget.RenderTarget, error) {
	texture, err := gtexture.New(width, height, create)
	if err != nil {
		return nil, err
	}
	return rendertarget.NewWithFramebuffer(texture, framebuffer), nil
}

func createFramebuffer(nativeTexture C.GLuint) C.GLuint {
	framebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &framebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, nativeTexture, 0)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) !=
		C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	return framebuffer
}

func NewFromImage(img image.Image) (*gtexture.Texture, error) {
	return gtexture.NewFromImage(img, createFromImage)
}
