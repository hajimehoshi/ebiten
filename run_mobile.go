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

//go:build android || ios
// +build android ios

package ebiten

// RunGameWithoutMainLoop runs the game, but doesn't call the loop on the main (UI) thread.
// RunGameWithoutMainLoop returns immediately unlike Run.
//
// Ebiten users should NOT call RunGameWithoutMainLoop.
// Instead, functions in github.com/hajimehoshi/ebiten/v2/mobile package calls this.
//
// TODO: Remove this. In order to remove this, the uiContext should be in another package.
func RunGameWithoutMainLoop(game Game) {
	theUIContext.set(game)
	uiDriver().RunWithoutMainLoop(theUIContext)
}
