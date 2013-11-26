package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
)

type OffscreenSetter struct {
	screenHeight int
	screenScale int
	usingMainFramebuffer bool
	projectionMatrix *[16]float32
}

func (s *OffscreenSetter) Set(framebuffer interface{}, x, y, width, height int) {
	f := framebuffer.(rendertarget.Framebuffer)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(f))
	err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER)
	if err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}

	C.glBlendFuncSeparate(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA,
		C.GL_ZERO, C.GL_ONE)

	C.glViewport(C.GLint(x), C.GLint(y),
		C.GLsizei(width), C.GLsizei(height))

	matrix := graphics.OrthoProjectionMatrix(x, width, y, height)
	if s.usingMainFramebuffer {
		actualScreenHeight := s.screenHeight * s.screenScale
		// Flip Y and move to fit with the top of the window.
		matrix[1][1] *= -1
		matrix[1][3] += float64(actualScreenHeight) / float64(height) * 2
	}

	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			s.projectionMatrix[i+j*4] = float32(matrix[i][j])
		}
	}
}
