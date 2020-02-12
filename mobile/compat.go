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

// +build android ios

package mobile

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/mobile/ebitenmobileview"
)

type game struct {
	update func(*ebiten.Image) error
	width  int
	height int
}

func (g *game) Update(screen *ebiten.Image) error {
	return g.update(screen)
}

func (g *game) Layout(viewWidth, viewHeight int) (screenWidth, screenHeight int) {
	return g.width, g.height
}

// Start is deprecated as of 1.10.0-alpha. Use `ebitenmobile` command and `SetGame` instead.
//
// Start starts the game and returns immediately.
//
// Different from ebiten.Run, this invokes only the game loop and not the main (UI) loop.
//
// The unit of width/height is device-independent pixel (dp on Android and point on iOS).
//
// Start is concurrent-safe.
//
// The argument title is ignored. This is for backward compatibility.
//
// Start always returns nil as of 1.5.0-alpha.
func Start(f func(*ebiten.Image) error, width, height int, scale float64, title string) error {
	ebitenmobileview.SetGame(&game{
		update: f,
		width:  width,
		height: height,
	})
	// As the view layout is already determined, ignore the layout calculation at ebitenmobileview.
	w := float64(width) * scale
	h := float64(height) * scale
	ebitenmobileview.Layout(w, h)
	return nil
}

// Update is deprecated as of 1.10.0-alpha. Use `ebitenmobile` command and `SetGame` instead.
//
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
	return ebitenmobileview.Update()
}

// UpdateTouchesOnAndroid is deprecated as of 1.10.0-alpha. Use `ebitenmobile` command instead.
//
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
	ebitenmobileview.UpdateTouchesOnAndroid(action, id, x, y)
}

// UpdateTouchesOnIOS is deprecated as of 1.10.0-alpha. Use `ebitenmobile` command instead.
//
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
	ebitenmobileview.UpdateTouchesOnIOS(phase, ptr, x, y)
}
