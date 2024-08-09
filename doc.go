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
//	// Game implements ebiten.Game interface.
//	type Game struct{}
//
//	// Update proceeds the game state.
//	// Update is called every tick (1/60 [s] by default).
//	func (g *Game) Update() error {
//	    // Write your game's logical update.
//	    return nil
//	}
//
//	// Draw draws the game screen.
//	// Draw is called every frame (typically 1/60[s] for 60Hz display).
//	func (g *Game) Draw(screen *ebiten.Image) {
//	    // Write your game's rendering.
//	}
//
//	// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
//	// If you don't have to adjust the screen size with the outside size, just return a fixed size.
//	func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
//	    return 320, 240
//	}
//
//	func main() {
//	    game := &Game{}
//	    // Specify the window size as you like. Here, a doubled size is specified.
//	    ebiten.SetWindowSize(640, 480)
//	    ebiten.SetWindowTitle("Your game's title")
//	    // Call ebiten.RunGame to start your game loop.
//	    if err := ebiten.RunGame(game); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// In the API document, 'the main thread' means the goroutine in init(), main() and their callees without 'go'
// statement. It is assured that 'the main thread' runs on the OS main thread. There are some Ebitengine functions (e.g.,
// DeviceScaleFactor) that must be called on the main thread under some conditions (typically, before ebiten.RunGame
// is called).
//
// # Environment variables
//
// `EBITENGINE_SCREENSHOT_KEY` environment variable specifies the key
// to take a screenshot. For example, if you run your game with
// `EBITENGINE_SCREENSHOT_KEY=q`, you can take a game screen's screenshot
// by pressing Q key. This works only on desktops and browsers.
//
// `EBITENGINE_INTERNAL_IMAGES_KEY` environment variable specifies the key
// to dump all the internal images. This is valid only when the build tag
// 'ebitenginedebug' is specified. This works only on desktops and browsers.
//
// `EBITENGINE_GRAPHICS_LIBRARY` environment variable specifies the graphics library.
// If the specified graphics library is not available, RunGame returns an error.
// This environment variable works when RunGame is called or RunGameWithOptions is called with GraphicsLibraryAuto.
// This can take one of the following value:
//
//	"auto":         Ebitengine chooses the graphics library automatically. This is the default value.
//	"opengl":       OpenGL, OpenGL ES, or WebGL.
//	"directx":      DirectX. This works only on Windows.
//	"metal":        Metal. This works only on macOS or iOS.
//	"playstation5": PlayStation 5. This works only on PlayStation 5.
//
// `EBITENGINE_DIRECTX` environment variable specifies various parameters for DirectX.
// You can specify multiple values separated by a comma. The default value is empty (i.e. no parameters).
//
//	"debug":                      Use a debug layer.
//	"warp":                       Use WARP (i.e. software rendering).
//	"version=VERSION":            Specify a DirectX version (e.g. 11).
//	"featurelevel=FEATURE_LEVEL": Specify a feature level (e.g. 11_0). This is for DirectX 12.
//
// The options taking arguments are exclusive, and if multiples are specified, the lastly specified value is adopted.
//
// The possible values for the option "version" are "11" and "12".
// If the version is not specified, the default version 11 is adopted.
// On Xbox, the "version" option is ignored and DirectX 12 is always adopted.
//
// The option "featurelevel" is valid only for DirectX 12.
// The possible values are "11_0", "11_1", "12_0", "12_1", and "12_2". The default value is "11_0".
//
// # Build tags
//
// `ebitenginedebug` outputs a log of graphics commands. This is useful to know what happens in Ebitengine. In general, the
// number of graphics commands affects the performance of your game.
//
// `ebitenginegldebug` enables a debug mode for OpenGL. This is valid only when the graphics library is OpenGL.
// This affects performance very much.
//
// `ebitenginesinglethread` disables Ebitengine's thread safety to unlock maximum performance. If you use this you will have
// to manage threads yourself. Functions like `SetWindowSize` will no longer be concurrent-safe with this build tag.
// They must be called from the main thread or the same goroutine as the given game's callback functions like Update
// `ebitenginesinglethread` works only with desktops and consoles.
// `ebitenginesinglethread` was deprecated as of v2.7. Use RunGameOptions.SingleThread instead.
//
// `microsoftgdk` is for Microsoft GDK (e.g. Xbox).
//
// `nintendosdk` is for NintendoSDK (e.g. Nintendo Switch).
//
// `nintendosdkprofile` enables a profiler for NintendoSDK.
//
// `playstation5` is for PlayStation 5.
package ebiten
