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

// NewImage returns an empty image.
func NewImage(width, height int, filter Filter) (*Image, error) {
	var innerImage *innerImage
	var err error
	ui.Use(func(c *opengl.Context) {
		var texture *graphics.Texture
		texture, err = graphics.NewTexture(c, width, height, glFilter(c, filter))
		if err != nil {
			return
		}
		innerImage, err = newInnerImage(c, texture)
		innerImage.Clear(c)
	})
	if err != nil {
		return nil, err
	}
	return &Image{inner: innerImage}, nil
}

// NewImageFromImage creates a new image with the given image (img).
func NewImageFromImage(img image.Image, filter Filter) (*Image, error) {
	var innerImage *innerImage
	var err error
	ui.Use(func(c *opengl.Context) {
		var texture *graphics.Texture
		texture, err = graphics.NewTextureFromImage(c, img, glFilter(c, filter))
		if err != nil {
			return
		}
		innerImage, err = newInnerImage(c, texture)
	})
	if err != nil {
		return nil, err
	}
	return &Image{inner: innerImage}, nil
}
