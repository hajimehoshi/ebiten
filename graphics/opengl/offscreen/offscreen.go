package offscreen

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
)

type Offscreen struct {
	screenHeight           int
	screenScale            int
	currentRenderTarget    *rendertarget.RenderTarget
	mainFramebufferTexture *rendertarget.RenderTarget
}

func New(screenWidth, screenHeight, screenScale int) *Offscreen {
	offscreen := &Offscreen{
		screenHeight: screenHeight,
		screenScale:  screenScale,
	}

	offscreen.mainFramebufferTexture = rendertarget.NewWithCurrentFramebuffer(
		screenWidth*screenScale,
		screenHeight*screenScale)

	return offscreen
}

func (o *Offscreen) Set(rt *rendertarget.RenderTarget) {
	o.currentRenderTarget = rt
	rt.SetAsViewport()
}

func (o *Offscreen) SetMainFramebuffer() {
	o.Set(o.mainFramebufferTexture)
}

func (o *Offscreen) DrawTexture(texture *texture.Texture,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	projectionMatrix := o.projectionMatrix()
	texture.Draw(projectionMatrix, geometryMatrix, colorMatrix)	
}

func (o *Offscreen) DrawTextureParts(texture *texture.Texture,
	parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	projectionMatrix := o.projectionMatrix()
	texture.DrawParts(parts, projectionMatrix, geometryMatrix, colorMatrix)
}

func (o *Offscreen) projectionMatrix() [16]float32 {
	matrix := o.currentRenderTarget.ProjectionMatrix()
	if o.currentRenderTarget == o.mainFramebufferTexture {
		actualScreenHeight := o.screenHeight * o.screenScale
		// Flip Y and move to fit with the top of the window.
		matrix[1][1] *= -1
		matrix[1][3] += float64(actualScreenHeight) /
			float64(graphics.AdjustSizeForTexture(actualScreenHeight)) * 2
	}

	projectionMatrix := [16]float32{}
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			projectionMatrix[i+j*4] = float32(matrix[i][j])
		}
	}
	return projectionMatrix
}
