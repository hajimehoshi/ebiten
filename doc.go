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
// You can start the game by calling the function RunGame.
//
//     // Game implements ebiten.Game interface.
//     type Game struct{}
//
//     // Update proceeds the game state.
//     // Update is called every tick (1/60 [s] by default).
//     func (g *Game) Update() error {
//         // Write your game's logical update.
//         return nil
//     }
//
//     // Draw draws the game screen.
//     // Draw is called every frame (typically 1/60[s] for 60Hz display).
//     func (g *Game) Draw(screen *ebiten.Image) {
//         // Write your game's rendering.
//     }
//
//     // Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
//     // If you don't have to adjust the screen size with the outside size, just return a fixed size.
//     func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
//         return 320, 240
//     }
//
//     func main() {
//         game := &Game{}
//         // Specify the window size as you like. Here, a doubled size is specified.
//         ebiten.SetWindowSize(640, 480)
//         ebiten.SetWindowTitle("Your game's title")
//         // Call ebiten.RunGame to start your game loop.
//         if err := ebiten.RunGame(game); err != nil {
//             log.Fatal(err)
//         }
//     }
//
// In the API document, 'the main thread' means the goroutine in init(), main() and their callees without 'go'
// statement. It is assured that 'the main thread' runs on the OS main thread. There are some Ebiten functions (e.g.,
// DeviceScaleFactor) that must be called on the main thread under some conditions (typically, before ebiten.RunGame
// is called).
//
// Environment variables
//
// `EBITEN_SCREENSHOT_KEY` environment variable specifies the key
// to take a screenshot. For example, if you run your game with
// `EBITEN_SCREENSHOT_KEY=q`, you can take a game screen's screenshot
// by pressing Q key. This works only on desktops.
//
// `EBITEN_INTERNAL_IMAGES_KEY` environment variable specifies the key
// to dump all the internal images. This is valid only when the build tag
// 'ebitendebug' is specified. This works only on desktops.
//
// Build tags
//
// `ebitendebug` outputs a log of graphics commands. This is useful to know what happens in Ebiten. In general, the
// number of graphics commands affects the performance of your game.
//
// `ebitengl` forces to use OpenGL in any environments.
//
// `ebitenwebgl1` forces to use WebGL 1 on browsers.
//
// `ebitensinglethread` disables Ebiten's thread safety to unlock maximum performance. If you use this you will have
// to manage threads yourself. Functions like IsKeyPressed will no longer be concurrent-safe with this build tag.
// They must be called from the main thread or the same goroutine as the given game's callback functions like Update
// to RunGame.
//
// `ebitenexternaldll` stops embedding DLL file in a Windows executable.
// `ebitenexternaldll` works only for Windows.
// The executable will require a DLL file at the working directory. Copy them from Ebiten repository's `internal/glfw`:
//
//   * `glfw_windows_386.dll` for Windows 386
//   * `glfw_windows_amd64.dll` for Windows amd64
//
// The directory path can be obtained by Go commands. For example, on PowerShell:
//
//   $dir = go list -f '{{.Dir}}' github.com/hajimehoshi/ebiten/v2/internal/glfw
//   echo $dir\glfw_windows_amd64.dll
//
// and on shell:
//
//   echo $(go list -f '{{.Dir}}' github.com/hajimehoshi/ebiten/v2/internal/glfw)/glfw_windows_amd64.dll
//
// Embedding a DLL and extracting it on the fly might be problematic on Windows since the application might be
// unexpectedly recognized as a virus by some virus checkers.
// `ebitenexternaldll` is useful for such cases. See #1832 for the discussion.
package ebiten
