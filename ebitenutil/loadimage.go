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

//go:build (darwin || freebsd || js || linux || windows) && !android && !ios
// +build darwin freebsd js linux windows
// +build !android
// +build !ios

package ebitenutil

import (
	"image"
	"io"

	"github.com/hajimehoshi/ebiten/v2"
)

// NewImageFromFile loads the file with path and returns ebiten.Image and image.Image.
//
// Image decoders must be imported when using NewImageFromFile. For example,
// if you want to load a PNG image, you'd need to add `_ "image/png"` to the import section.
//
// How to solve path depends on your environment. This varies on your desktop or web browser.
// Note that this doesn't work on mobiles.
//
// For productions, instead of using NewImageFromFile, it is safer to embed your resources with go:embed.
func NewImageFromFile(path string) (*ebiten.Image, image.Image, error) {
	file, err := OpenFile(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	return NewImageFromReader(file)
}

// NewImageFromReader loads from the io.Reader and returns ebiten.Image and image.Image.
//
// Image decoders must be imported when using NewImageFromReader. For example,
// if you want to load a PNG image, you'd need to add `_ "image/png"` to the import section.
func NewImageFromReader(reader io.Reader) (*ebiten.Image, image.Image, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, nil, err
	}
	img2 := ebiten.NewImageFromImage(img)
	return img2, img, err
}
