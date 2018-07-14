// Copyright 2018 The Ebiten Authors
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
	"image"
	"net/http"

	"github.com/hajimehoshi/ebiten"
)

// NewImageFromURL creates a new ebiten.Image from the given URL.
//
// Image decoders must be imported when using NewImageFromURL. For example,
// if you want to load a PNG image, you'd need to add `_ "image/png"` to the import section.
//
// FilterDefault is used at NewImgeFromImage internally.
func NewImageFromURL(url string) (*ebiten.Image, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, err
	}

	eimg, _ := ebiten.NewImageFromImage(img, ebiten.FilterDefault)
	return eimg, nil
}
