// Copyright 2021 Hajime Hoshi
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

//go:build (darwin || freebsd || js || linux || windows || android || ios) && go1.16
// +build darwin freebsd js linux windows android ios
// +build go1.16

package ebitenutil

import (
	"image"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
)

// NewImageFromFS loads the file from a fs.FS filesystem with path and returns ebiten.Image and image.Image.
//
// go1.16+ only
//
// Image decoders must be imported when using NewImageFromFS. For example,
// if you want to load a PNG image, you'd need to add `_ "image/png"` to the import section.
//
// This is useful if you've ebedded a filesystem with embed.FS.
func NewImageFromFS(filesystem fs.FS, path string) (*ebiten.Image, image.Image, error) {
	file, err := filesystem.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, nil, err
	}
	img2 := ebiten.NewImageFromImage(img)
	return img2, img, err
}
