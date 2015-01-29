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

// Package ebiten provides graphics and input API to develop a 2D game.
//
// You can start the game by calling the function Run.
//
//     func update(screen *ebiten.Image) error {
//         // Define your game.
//     }
//
//     func main() {
//         ebiten.Run(update, 320, 240, 2, "Your game's title")
//     }
package ebiten
