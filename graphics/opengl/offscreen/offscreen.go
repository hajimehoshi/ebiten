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
	gtexture "github.com/hajimehoshi/go-ebiten/graphics/texture"
)

type Offscreen struct {
	screenHeight           int
	screenScale            int
	mainFramebufferTexture *gtexture.RenderTarget
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

func (o *Offscreen) Set(rt *gtexture.RenderTarget) {
	C.glFlush()
	rt.SetAsOffscreen(&setter{o, rt == o.mainFramebufferTexture})
}

func (o *Offscreen) SetMainFramebuffer() {
	o.Set(o.mainFramebufferTexture)
}

func (o *Offscreen) DrawTexture(texture *gtexture.Texture,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture.Draw(&drawable{o, geometryMatrix, colorMatrix})
}

func (o *Offscreen) DrawTextureParts(texture *gtexture.Texture,
	parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture.DrawParts(parts, &drawable{o, geometryMatrix, colorMatrix})
}

type setter struct {
	offscreen            *Offscreen
	usingMainFramebuffer bool
}

func (s *setter) Set(framebuffer interface{}, x, y, width, height int) {
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
		actualScreenHeight := s.offscreen.screenHeight * s.offscreen.screenScale
		// Flip Y and move to fit with the top of the window.
		matrix[1][1] *= -1
		matrix[1][3] += float64(actualScreenHeight) / float64(height) * 2
	}

	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			s.offscreen.projectionMatrix[i+j*4] = float32(matrix[i][j])
		}
	}
}

type drawable struct {
	offscreen      *Offscreen
	geometryMatrix matrix.Geometry
	colorMatrix    matrix.Color
}

func (d *drawable) Draw(native interface{}, quads []graphics.TextureQuad) {
	shader.DrawTexture(native.(texture.Native),
		d.offscreen.projectionMatrix, quads,
		d.geometryMatrix, d.colorMatrix)
}
