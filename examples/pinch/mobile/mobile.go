// Copyright 2026 The Ebitengine Authors
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

// Package mobile is the entry point of the example for Android and iOS.
//
// Build a library with ebitenmobile, for example:
//
//	go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile bind -target android -javapkg com.example.pinch -o pinch.aar ./examples/pinch/mobile
//	go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile bind -target ios -o Pinch.xcframework ./examples/pinch/mobile
//
// Then put EbitenView (Android) or EbitenViewController (iOS) in an app.
// See https://ebitengine.org/en/documents/mobile.html.
package mobile

import (
	ebitenmobile "github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/hajimehoshi/ebiten/v2/examples/pinch/pinch"
)

func init() {
	ebitenmobile.SetGame(pinch.NewGame())
}

// Dummy is a dummy exported function.
//
// gomobile doesn't compile a package that doesn't include any exported function.
// Dummy forces gomobile to compile this package.
func Dummy() {}
