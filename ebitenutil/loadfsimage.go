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

package ebitenutil

import (
	"io/fs"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// NewImageFromEmbedFile loads the file from embed.FS with path and returns ebiten.Image and image.Image.
//
// Image decoders must be imported when using NewImageFromEmbedFile. For example,
// if you want to load a PNG image, you'd need to add `_ "image/png"` to the import section.
func NewImageFromFileSystem(fsFs fs.FS, path string) (*ebiten.Image, image.Image, error) {
	file, err := fsFs.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	return NewImageFromReader(file)
}
