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

// Package mobile provides functions for mobile platforms (Android and iOS).
//
// This package is used when you use `gomobile bind`.
// For `gomobile build`, you don't have to use this package.
//
// For usage, see https://github.com/hajimehoshi/ebiten/wiki/Mobile, https://github.com/hajimehoshi/ebiten/wiki/Android and https://github.com/hajimehoshi/ebiten/wiki/iOS.
package mobile

import (
	"sync"

	"github.com/hajimehoshi/ebiten"
)

var (
	// mobileMutex is a mutex required for each function.
	// For example, on Android, Update can be called from a different thread:
	// https://developer.android.com/reference/android/opengl/GLSurfaceView.Renderer
	mobileMutex sync.Mutex
)

// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
//
// The unit of width/height is device-independent pixel (dp on Android and point on iOS).
//
// Start is concurrent-safe.
//
// Start always returns nil as of 1.5.0-alpha.
func Start(f func(*ebiten.Image) error, width, height int, scale float64, title string) error {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	start(f, width, height, scale, title)
	return nil
}

// Update updates and renders the game.
// This should be called on every frame.
//
// If Update is called before Start is called, Update panics.
//
// On Android, this should be called at onDrawFrame of Renderer (used by GLSurfaceView).
//
// On iOS, this should be called at glkView:drawInRect: of GLKViewDelegate.
//
// Update is concurrent-safe.
//
// Update returns error when 1) OpenGL error happens, or 2) f in Start returns error samely as ebiten.Run.
func Update() error {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	return update()
}

// UpdateTouchesOnAndroid updates the touch state on Android.
//
// This should be called with onTouchEvent of GLSurfaceView like this:
//
//     private double mDeviceScale = 0.0;
//
//     // pxToDp converts an value in pixels to dp.
//     private double pxToDp(double x) {
//         if (mDeviceScale == 0.0) {
//             mDeviceScale = getResources().getDisplayMetrics().density;
//         }
//         return x / mDeviceScale;
//     }
//
//     @Override
//     public boolean onTouchEvent(MotionEvent e) {
//         for (int i = 0; i < e.getPointerCount(); i++) {
//             int id = e.getPointerId(i);
//             int x = (int)e.getX(i);
//             int y = (int)e.getY(i);
//             // Exported function for UpdateTouchesOnAndroid
//             YourGame.UpdateTouchesOnAndroid(e.getActionMasked(), id, (int)pxToDp(x), (int)pxToDp(y));
//         }
//         return true;
//     }
//
// The coodinate x/y is in dp.
//
// UpdateTouchesOnAndroid can be called even before Start is called.
//
// UpdateTouchesOnAndroid is concurrent-safe.
//
// For more details, see https://github.com/hajimehoshi/ebiten/wiki/Android.
func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	updateTouchesOnAndroid(action, id, x, y)
}

// UpdateTouchesOnIOS updates the touch state on iOS.
//
// This should be called with touch handlers of UIViewController like this:
//
//     - (GLKView*)glkView {
//         return (GLKView*)[self.view viewWithTag:100];
//     }
//     - (void)updateTouches:(NSSet*)touches {
//         for (UITouch* touch in touches) {
//             if (touch.view != [self glkView]) {
//                 continue;
//             }
//             CGPoint location = [touch locationInView: [self glkView]];
//             // Exported function for UpdateTouchesOnIOS
//             YourGameUpdateTouchesOnIOS(touch.phase, (int64_t)touch, location.x, location.y);
//         }
//     }
//     - (void)touchesBegan:(NSSet*)touches withEvent:(UIEvent*)event {
//         [self updateTouches:touches];
//     }
//     - (void)touchesMoved:(NSSet*)touches withEvent:(UIEvent*)event {
//         [self updateTouches:touches];
//     }
//     - (void)touchesEnded:(NSSet*)touches withEvent:(UIEvent*)event {
//         [self updateTouches:touches];
//     }
//     - (void)touchesCancelled:(NSSet*)touches withEvent:(UIEvent*)event {
//         [self updateTouches:touches];
//     }
//
// The coodinate x/y is in point.
//
// UpdateTouchesOnIOS can be called even before Start is called.
//
// UpdateTouchesOnIOS is concurrent-safe.
//
// For more details, see https://github.com/hajimehoshi/ebiten/wiki/iOS.
func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	mobileMutex.Lock()
	defer mobileMutex.Unlock()

	updateTouchesOnIOSImpl(phase, ptr, x, y)
}
