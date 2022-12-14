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
// For usage, see https://ebitengine.org/en/documents/mobile.html.
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// SetGame sets a mobile game.
//
// SetGame is expected to be called only once.
//
// SetGame can be called anytime. Until SetGame is called, the game does not start.
func SetGame(game ebiten.Game) {
	SetGameWithOptions(game, nil)
}

// SetGameWithOptions sets a mobile game with the specified options.
//
// SetGameWithOptions is expected to be called only once.
//
// SetGameWithOptions can be called anytime. Until SetGameWithOptions is called, the game does not start.
func SetGameWithOptions(game ebiten.Game, options *ebiten.RunGameOptions) {
	setGame(game, options)
}
