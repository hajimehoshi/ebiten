/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"image"
)

type Game interface {
	Update() error
	Draw(gr GraphicsContext) error
}

func IsKeyPressed(key Key) bool {
	return currentUI.canvas.input.IsKeyPressed(key)
}

func CursorPosition() (x, y int) {
	return currentUI.canvas.input.CursorPosition()
}

func IsMouseButtonPressed(mouseButton MouseButton) bool {
	return currentUI.canvas.input.IsMouseButtonPressed(mouseButton)
}

func NewRenderTargetID(width, height int, filter Filter) (RenderTargetID, error) {
	return currentUI.canvas.NewRenderTargetID(width, height, filter)
}

func NewTextureID(img image.Image, filter Filter) (TextureID, error) {
	return currentUI.canvas.NewTextureID(img, filter)
}
