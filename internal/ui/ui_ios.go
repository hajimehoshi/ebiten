// Copyright 2022 The Ebiten Authors
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

package ui

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Foundation -framework UIKit
//
// #import <UIKit/UIKit.h>
//
// #cgo noescape displayInfoOnMainThread
// #cgo nocallback displayInfoOnMainThread
// static void displayInfoOnMainThread(float* width, float* height, float* scale, uintptr_t viewPtr) {
//   *width = 0;
//   *height = 0;
//   *scale = 1;
//   UIView* view = (__bridge UIView*)(void*)viewPtr;
//   UIWindow* window = view.window;
//   if (!window) {
//     return;
//   }
//   UIWindowScene* scene = window.windowScene;
//   if (!scene) {
//     return;
//   }
//   CGRect bounds = scene.screen.bounds;
//   *width = bounds.size.width;
//   *height = bounds.size.height;
//   *scale = scene.screen.nativeScale;
// }
import "C"

import (
	"errors"
	"fmt"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverCreatorImpl struct {
	colorSpace color.ColorSpace
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	m, err1 := g.newMetal()
	if err1 == nil {
		return m, GraphicsLibraryMetal, nil
	}
	o, err2 := g.newOpenGL()
	if err2 == nil {
		return o, GraphicsLibraryOpenGL, nil
	}
	return nil, GraphicsLibraryUnknown, fmt.Errorf("ui: failed to choose graphics drivers: Metal: %v, OpenGL: %v", err1, err2)
}

func (g *graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: DirectX is not supported in this environment")
}

func (g *graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return metal.NewGraphics(g.colorSpace)
}

func (*graphicsDriverCreatorImpl) newPlayStation5() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: PlayStation 5 is not supported in this environment")
}

// SetUIView sets the view the game is rendered into. It must be called
// whichever graphics library is used.
func (u *UserInterface) SetUIView(uiview uintptr) error {
	u.uiView.Store(uiview)
	u.refreshDisplayInfo()
	select {
	case err := <-u.errCh:
		return err
	case <-u.graphicsLibraryInitCh:
	}

	// Only the Metal driver needs the view. The OpenGL driver renders through
	// the context the view owns.
	if g, ok := u.graphicsDriver.(interface{ SetUIView(uintptr) }); ok {
		g.SetUIView(uiview)
	}
	return nil
}

// UIView returns the UIView pointer set by [UserInterface.SetUIView], or 0.
func (u *UserInterface) UIView() uintptr {
	return u.uiView.Load()
}

func (u *UserInterface) IsGL() (bool, error) {
	select {
	case err := <-u.errCh:
		return false, err
	case <-u.graphicsLibraryInitCh:
	}

	return u.GraphicsLibrary() == GraphicsLibraryOpenGL, nil
}

func dipToNativePixels(x float64, scale float64) float64 {
	return x
}

func dipFromNativePixels(x float64, scale float64) float64 {
	return x
}

// refreshDisplayInfo records the display info for displayInfo to serve on any
// thread. refreshDisplayInfo must be called on the main thread.
func (u *UserInterface) refreshDisplayInfo() {
	view := u.uiView.Load()
	if view == 0 {
		return
	}

	var cWidth, cHeight, cScale C.float
	C.displayInfoOnMainThread(&cWidth, &cHeight, &cScale, C.uintptr_t(view))
	theDisplayInfo.Store(&displayInfoValues{
		width:  float64(cWidth),
		height: float64(cHeight),
		scale:  float64(cScale),
	})
}

func (u *UserInterface) RunOnMainThread(f func()) {
	b := objc.NewBlock(func(_ objc.Block) {
		f()
	})
	defer b.Release()
	dispatchSync(dispatchMainQ, b)
}
