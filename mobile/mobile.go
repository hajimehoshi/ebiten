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
	"github.com/hajimehoshi/ebiten/internal/ui"
)

type EventDispatcher interface {
	Render() error

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
	//             YourGame.CurrentEventDispatcher().UpdateTouchesOnAndroid(e.getActionMasked(), id, x, y);
	//         }
	//         return true;
	//     }
	UpdateTouchesOnAndroid(action int, id int, x, y int)
	UpdateTouchesOnIOS(phase int, ptr int, x, y int)
}

// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
func Start(f func(*ebiten.Image) error, width, height int, scale float64, title string) (EventDispatcher, error) {
	return start(f, width, height, scale, title)
}

type position struct {
	x int
	y int
}

type eventDispatcher struct {
	touches map[int]position
}

func (e *eventDispatcher) Render() error {
	return render()
}

// touch implements ui.Touch.
type touch struct {
	id       int
	position position
}

func (t touch) ID() int {
	return t.id
}

func (t touch) Position() (int, int) {
	// TODO: Is this OK to adjust the position here?
	return int(float64(t.position.x) / ui.CurrentUI().ScreenScale()),
		int(float64(t.position.y) / ui.CurrentUI().ScreenScale())
}

func (e *eventDispatcher) UpdateTouchesOnAndroid(action int, id int, x, y int) {
	switch action {
	case 0x00, 0x05, 0x02: // ACTION_DOWN, ACTION_POINTER_DOWN, ACTION_MOVE
		e.touches[id] = position{x, y}
		e.updateTouches()
	case 0x01, 0x06: // ACTION_UP, ACTION_POINTER_UP
		delete(e.touches, id)
		e.updateTouches()
	}
}

func (e *eventDispatcher) UpdateTouchesOnIOS(phase int, ptr int, x, y int) {
	e.updateTouchesOnIOSImpl(phase, ptr, x, y)
}
