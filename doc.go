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
//     // update is called every frame (1/60 [s]).
//     func update(screen *ebiten.Image) error {
//
//         // Write your game's logical update.
//
//         if IsRunningSlowly() {
//             // When the game is running slowly, the rendering result
//             // will not be adopted.
//             return nil
//         }
//
//         // Write your game's rendering.
//
//         return nil
//     }
//
//     func main() {
//         // Call ebiten.Run to start your game loop.
//         ebiten.Run(update, 320, 240, 2, "Your game's title")
//     }
//
// The EBITEN_SCREENSHOT_KEY environment variable specifies the key
// to take a screenshot. For example, if you run your game with
// `EBITEN_SCREENSHOT_KEY=q`, you can take a game screen's screenshot
// by pressing Q key.
//
// The EBITEN_INTERNAL_IMAGES_KEY environment variable specifies the key
// to dump all the internal images.
package ebiten
