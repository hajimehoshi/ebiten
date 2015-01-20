// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"image"
)

// IsKeyPressed returns a boolean indicating whether key is pressed.
func IsKeyPressed(key Key) bool {
	return ui.IsKeyPressed(ui.Key(key))
}

// CursorPosition returns a position of a mouse cursor.
func CursorPosition() (x, y int) {
	return ui.CursorPosition()
}

// IsMouseButtonPressed returns a boolean indicating whether mouseButton is pressed.
func IsMouseButtonPressed(mouseButton MouseButton) bool {
	return ui.IsMouseButtonPressed(ui.MouseButton(mouseButton))
}

// GamepadAxisNum returns the number of axes of the gamepad.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func GamepadAxisNum(id int) int {
	return ui.GamepadAxisNum(id)
}

// GamepadAxis returns the float value [-1.0 - 1.0] of the axis.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func GamepadAxis(id int, axis int) float64 {
	return ui.GamepadAxis(id, axis)
}

// GamepadButtonNum returns the number of the buttons of the gamepad.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func GamepadButtonNum(id int) int {
	return ui.GamepadButtonNum(id)
}

// IsGamepadButtonPressed returns the boolean indicating the buttons is pressed or not.
//
// NOTE: Gamepad API is available only on desktops, Chrome and Firefox.
// To use this API, browsers might require rebooting the browser.
func IsGamepadButtonPressed(id int, button GamepadButton) bool {
	return ui.IsGamepadButtonPressed(id, ui.GamepadButton(button))
}

// NewImage returns an empty image.
//
// NewImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
func NewImage(width, height int, filter Filter) (*Image, error) {
	var img *Image
	var err error
	ui.Use(func(c *opengl.Context) {
		var texture *graphics.Texture
		var framebuffer *graphics.Framebuffer
		texture, err = graphics.NewTexture(c, width, height, glFilter(c, filter))
		if err != nil {
			return
		}
		framebuffer, err = graphics.NewFramebufferFromTexture(c, texture)
		if err != nil {
			return
		}
		img = &Image{framebuffer: framebuffer, texture: texture}
	})
	if err != nil {
		return nil, err
	}
	if err := img.Clear(); err != nil {
		return nil, err
	}
	return img, nil
}

// NewImageFromImage creates a new image with the given image (img).
//
// NewImageFromImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
func NewImageFromImage(img image.Image, filter Filter) (*Image, error) {
	var eimg *Image
	var err error
	ui.Use(func(c *opengl.Context) {
		var texture *graphics.Texture
		var framebuffer *graphics.Framebuffer
		texture, err = graphics.NewTextureFromImage(c, img, glFilter(c, filter))
		if err != nil {
			return
		}
		framebuffer, err = graphics.NewFramebufferFromTexture(c, texture)
		if err != nil {
			return
		}
		eimg = &Image{framebuffer: framebuffer, texture: texture}
	})
	if err != nil {
		return nil, err
	}
	return eimg, nil
}
