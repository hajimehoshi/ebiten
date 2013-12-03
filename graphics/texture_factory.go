package graphics

import (
	"image"
)

// TODO: Remove this?
type TextureFactory interface {
	CreateRenderTarget(width, height int) (RenderTargetId, error)
	CreateTextureFromImage(img image.Image) (TextureId, error)
}
