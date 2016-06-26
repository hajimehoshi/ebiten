// Copyright 2016 Hajime Hoshi
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

package mobile

import (
	"github.com/hajimehoshi/ebiten"
)

// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
func Start(f func(*ebiten.Image) error, width, height int, scale float64, title string) error {
	return start(f, width, height, scale, title)
}

// Render updates and renders the game.
//
// This should be called on every frame.
//
// On Android, this should be called at onDrawFrame of Renderer (used by GLSurfaceView).
//
// On iOS, this should be called at glkView:drawInRect: of GLKViewDelegate.
func Render() error {
	return render()
}

// UpdateTouchesOnAndroid updates the touch state on Android.
//
// This should be called with onTouchEvent of GLSurfaceView like this:
//
//     @Override
//     public boolean onTouchEvent(MotionEvent e) {
//         for (int i = 0; i < e.getPointerCount(); i++) {
//             int id = e.getPointerId(i);
//             int x = (int)e.getX(i);
//             int y = (int)e.getY(i);
//             YourGame.UpdateTouchesOnAndroid(e.getActionMasked(), id, x, y);
//         }
//         return true;
//     }
func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	updateTouchesOnAndroid(action, id, x, y)
}

func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	updateTouchesOnIOSImpl(phase, ptr, x, y)
}
