// Copyright 2019 The Ebiten Authors
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

// Package mobile provides functions for mobile platforms (Android and iOS).
//
// This package is used when you use `ebitenmobile bind`.
//
// For usage, see https://ebiten.org/mobile.html.
package mobile

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/mobile/ebitenmobileview"
)

// Game defines necessary functions for a mobile game.
type Game interface {
	// Update updates a game by one frame.
	Update(*ebiten.Image) error

	// Layout accepts a native view size in DP (device-independent pixels) and returns the game's logical screen
	// size.
	//
	// The screen scale is automatically adjusted to fit the view.
	//
	// Layout is called at an initialization and whenever the view size is changed.
	//
	// You can return a fixed screen size if you don't care, or you can also return a calculated screen size
	// adjusted with the given view size.
	Layout(viewWidth, viewHeight int) (screenWidth, screenHeight int)
}

// SetGame sets a mobile game.
//
// SetGame is epxected to be called only once.
//
// SetGame can be called anytime. Until SetGame is called, the game does not start.
func SetGame(game Game) {
	ebitenmobileview.SetGame(game)
}
