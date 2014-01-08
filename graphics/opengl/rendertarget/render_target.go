package rendertarget

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
)

type Framebuffer C.GLuint

type RenderTarget struct {
	framebuffer Framebuffer
	width  int
	height int
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

func (r *RenderTarget) SetAsViewport() {
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(r.framebuffer))
	err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER)
	if err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}

	C.glBlendFuncSeparate(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA,
		C.GL_ZERO, C.GL_ONE)

	width := graphics.AdjustSizeForTexture(r.width)
	height := graphics.AdjustSizeForTexture(r.height)
	C.glViewport(0, 0, C.GLsizei(width), C.GLsizei(height))
}

func (r *RenderTarget) ProjectionMatrix() [4][4]float64 {
	width := graphics.AdjustSizeForTexture(r.width)
	height := graphics.AdjustSizeForTexture(r.height)
	return graphics.OrthoProjectionMatrix(0, width, 0, height)
}

func (r *RenderTarget) Dispose() {
	f := C.GLuint(r.framebuffer)
	C.glDeleteFramebuffers(1, &f)
}
