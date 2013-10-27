package rendertarget

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
	"github.com/hajimehoshi/go-ebiten/graphics/rendertarget"
)

type Framebuffer C.GLuint

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

func NewRenderTarget(width, height int) (*rendertarget.RenderTarget, error) {
	tex, err := texture.New(width, height)
	if err != nil {
		return nil, err
	}
	framebuffer := createFramebuffer(C.GLuint(tex.Native().(texture.Native)))
	return rendertarget.NewWithFramebuffer(tex, Framebuffer(framebuffer)), nil
}

func NewRenderTargetWithFramebuffer(width, height int, framebuffer Framebuffer) (*rendertarget.RenderTarget, error) {
	tex, err := texture.New(width, height)
	if err != nil {
		return nil, err
	}
	return rendertarget.NewWithFramebuffer(tex, framebuffer), nil
}
