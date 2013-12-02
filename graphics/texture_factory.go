package graphics

import (
	"image"
)

type TextureFactory interface {
	CreateRenderTarget(width, height int) (RenderTargetId, error)
	CreateTextureFromImage(img image.Image) (TextureId, error)
}
