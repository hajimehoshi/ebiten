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

type RenderTarget struct {
	framebuffer texture.Framebuffer
	width       int
	height      int
}

func Create(width, height int, filter graphics.Filter) (
	*RenderTarget, *texture.Texture, error) {
	tex, err := texture.Create(width, height, filter)
	if err != nil {
		return nil, nil, err
	}
	framebuffer := tex.CreateFramebuffer()
	return &RenderTarget{framebuffer, width, height}, tex, nil
}

func CreateWithFramebuffer(width, height int, framebuffer texture.Framebuffer) (
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
