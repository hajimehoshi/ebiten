package dummy

import (
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/ui"
	"image"
)

type TextureFactory struct{}

func (t *TextureFactory) CreateRenderTarget(width, height int, filter graphics.Filter) (graphics.RenderTargetId, error) {
	return 0, nil
}

func (t *TextureFactory) CreateTexture(img image.Image, filter graphics.Filter) (graphics.TextureId, error) {
	return 0, nil
}

type UI struct{}

func (u *UI) CreateCanvas(widht, height, scale int, title string) ui.Canvas {
	return &Canvas{}
}

func (u *UI) Start() {
}

func (u *UI) DoEvents() {
}

func (u *UI)Terminate() {
}

type Keys struct{}

func (k *Keys) Includes(key ui.Key) bool {
	return false
}

type InputState struct{}

func (i *InputState) PressedKeys() ui.Keys {
	return &Keys{}
}

func (i *InputState) MouseX() int {
	return -1
}

func (i *InputState) MouseY() int {
	return -1
}

type Canvas struct{}

func (c *Canvas) Draw(func(graphics.Context)) {
}

func (c *Canvas) IsClosed() bool {
	return true
}

func (c *Canvas) InputState() ui.InputState {
	return &InputState{}
}
