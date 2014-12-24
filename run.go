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

package ebiten

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
//
// This function must be called from the main thread.
func Run(f func(*Image) error, width, height, scale int, title string) error {
	err := startUI(width, height, scale, title)
	if err != nil {
		return err
	}
	ui := currentUI
	defer ui.terminate()

	currentUI = ui
	defer func() {
		currentUI = nil
	}()

	for {
		ui.doEvents()
		if ui.isClosed() {
			return nil
		}
		if err := ui.draw(f); err != nil {
			return err
		}
	}
}
