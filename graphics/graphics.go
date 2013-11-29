package graphics

import (
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
)

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

type TexturePart struct {
	LocationX int
	LocationY int
	Source    Rect
}

type Canvas interface {
	ToTexture(id RenderTargetId) TextureId

	Clear()
	Fill(r, g, b uint8)
	DrawTexture(id TextureId,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(id TextureId,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	ResetOffscreen()
	SetOffscreen(id RenderTargetId)
}

type TextureFactory interface {
	CreateRenderTarget(width, height int) (RenderTargetId, error)
	CreateTextureFromImage(img image.Image) (TextureId, error)
}

type TextureId int

// A render target is essentially same as a texture, but it is assumed that the
// all alpha of a render target is maximum.
type RenderTargetId int

func OrthoProjectionMatrix(left, right, bottom, top int) [4][4]float64 {
	e11 := float64(2) / float64(right-left)
	e22 := float64(2) / float64(top-bottom)
	e14 := -1 * float64(right+left) / float64(right-left)
	e24 := -1 * float64(top+bottom) / float64(top-bottom)

	return [4][4]float64{
		{e11, 0, 0, e14},
		{0, e22, 0, e24},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}
