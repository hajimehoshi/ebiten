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

// +build darwin,!arm,!arm64 linux windows
// +build !android
// +build !ios

package ebitenutil

import (
	"image"

	"github.com/hajimehoshi/ebiten"
)

// NewImageFromFile loads the file path and returns ebiten.Image and image.Image.
//
// The current directory for path depends on your environment. This will vary on your desktop or web browser.
// Note that this doesn't work on browsers.
// It'll be safer to embed your resource, e.g., with github.com/jteeuwen/go-bindata instead of using this function.
func NewImageFromFile(path string, filter ebiten.Filter) (*ebiten.Image, image.Image, error) {
	file, err := OpenFile(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, nil, err
	}
	img2, err := ebiten.NewImageFromImage(img, filter)
	if err != nil {
		return nil, nil, err
	}
	return img2, img, err
}
