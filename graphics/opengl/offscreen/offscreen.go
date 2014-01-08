package offscreen

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/shader"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
)

type Offscreen struct {
	screenHeight           int
	screenScale            int
	mainFramebufferTexture *rendertarget.RenderTarget
	projectionMatrix       [16]float32
}

func New(screenWidth, screenHeight, screenScale int) *Offscreen {
	offscreen := &Offscreen{
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	mainFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &mainFramebuffer)

	var err error
	offscreen.mainFramebufferTexture, err = rendertarget.CreateWithFramebuffer(
		screenWidth*screenScale,
		screenHeight*screenScale,
		rendertarget.Framebuffer(mainFramebuffer))
	if err != nil {
		panic("creating main framebuffer failed: " + err.Error())
	}

	return offscreen
}

func (o *Offscreen) Set(rt *rendertarget.RenderTarget) {
	C.glFlush()
	// TODO: Calc x, y, width, heigth at another function
	o.doSet(rt.Framebuffer, 0, 0,
		graphics.AdjustSizeForTexture(rt.Width), graphics.AdjustSizeForTexture(rt.Height))
}

func (o *Offscreen) SetMainFramebuffer() {
	o.Set(o.mainFramebufferTexture)
}

func (o *Offscreen) DrawTexture(texture *texture.Texture,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	quad := graphics.TextureQuadForTexture(texture.Width, texture.Height)
	shader.DrawTexture(texture.Native,
		o.projectionMatrix, []graphics.TextureQuad{quad},
		geometryMatrix, colorMatrix)
}

func (o *Offscreen) DrawTextureParts(texture *texture.Texture,
	parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	quads := graphics.TextureQuadsForTextureParts(parts, texture.Width, texture.Height)
	shader.DrawTexture(texture.Native,
		o.projectionMatrix, quads,
		geometryMatrix, colorMatrix)
}

func (o *Offscreen) doSet(framebuffer rendertarget.Framebuffer, x, y, width, height int) {
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(framebuffer))
	err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER)
	if err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}

	C.glBlendFuncSeparate(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA,
		C.GL_ZERO, C.GL_ONE)

	C.glViewport(C.GLint(x), C.GLint(y),
		C.GLsizei(width), C.GLsizei(height))

	matrix := graphics.OrthoProjectionMatrix(x, width, y, height)
	if framebuffer == o.mainFramebufferTexture.Framebuffer {
		actualScreenHeight := o.screenHeight * o.screenScale
		// Flip Y and move to fit with the top of the window.
		matrix[1][1] *= -1
		matrix[1][3] += float64(actualScreenHeight) / float64(height) * 2
	}

	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			o.projectionMatrix[i+j*4] = float32(matrix[i][j])
		}
	}
}
