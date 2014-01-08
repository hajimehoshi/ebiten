package rendertarget

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
)

type Framebuffer C.GLuint

type RenderTarget struct {
	Framebuffer
	Width  int
	Height int
}

func createFramebuffer(nativeTexture C.GLuint) Framebuffer {
	framebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &framebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)

	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	defer C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))

	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, nativeTexture, 0)
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) !=
		C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	// Set this framebuffer opaque because alpha values on a target might be
	// confusing.
	C.glClearColor(0, 0, 0, 1)
	C.glClear(C.GL_COLOR_BUFFER_BIT)

	return Framebuffer(framebuffer)
}

func Create(width, height int, filter graphics.Filter) (
	*RenderTarget, *texture.Texture, error) {
	tex, err := texture.Create(width, height, filter)
	if err != nil {
		return nil, nil, err
	}
	framebuffer := createFramebuffer(C.GLuint(tex.Native))
	return &RenderTarget{framebuffer, tex.Width, tex.Height}, tex, nil
}

func CreateWithFramebuffer(width, height int, framebuffer Framebuffer) (
	*RenderTarget, error) {
	return &RenderTarget{framebuffer, width, height}, nil
}

func (r *RenderTarget) Dispose() {
	f := C.GLuint(r.Framebuffer)
	C.glDeleteFramebuffers(1, &f)
}
