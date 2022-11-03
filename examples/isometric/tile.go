// Copyright 2021 The Ebiten Authors
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

package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Tile represents a space with an x,y coordinate within a Level. Any number of
// sprites may be added to a Tile.
type Tile struct {
	sprites []*ebiten.Image
}

// AddSprite adds a sprite to the Tile.
func (t *Tile) AddSprite(s *ebiten.Image) {
	t.sprites = append(t.sprites, s)
}

// Draw draws the Tile on the screen using the provided options.
func (t *Tile) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	for _, s := range t.sprites {
		screen.DrawImage(s, options)
	}
}
