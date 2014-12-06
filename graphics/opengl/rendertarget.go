package opengl

import (
	"fmt"
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
)

type RenderTarget struct {
	framebuffer gl.Framebuffer
	width       int
	height      int
	flipY       bool
}

func createFramebuffer(nativeTexture gl.Texture) gl.Framebuffer {
	framebuffer := gl.GenFramebuffer()
	framebuffer.Bind()

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0,
		gl.TEXTURE_2D, nativeTexture, 0)
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	// Set this framebuffer opaque because alpha values on a target might be
	// confusing.
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	return framebuffer
}

func (r *RenderTarget) setAsViewport() {
	gl.Flush()
	r.framebuffer.Bind()
	err := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if err != gl.FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}

	gl.BlendFuncSeparate(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA, gl.ZERO, gl.ONE)

	width := graphics.AdjustSizeForTexture(r.width)
	height := graphics.AdjustSizeForTexture(r.height)
	gl.Viewport(0, 0, width, height)
}

func (r *RenderTarget) projectionMatrix() [4][4]float64 {
	width := graphics.AdjustSizeForTexture(r.width)
	height := graphics.AdjustSizeForTexture(r.height)
	matrix := graphics.OrthoProjectionMatrix(0, width, 0, height)
	if r.flipY {
		matrix[1][1] *= -1
		matrix[1][3] += float64(r.height) /
			float64(graphics.AdjustSizeForTexture(r.height)) * 2
	}
	return matrix
}

func (r *RenderTarget) dispose() {
	r.framebuffer.Delete()
}
