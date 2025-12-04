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

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UIKit

#import <UIKit/UIKit.h>

static void displayInfoOnMainThread(float* width, float* height, float* scale, UIView* view) {
  *width = 0;
  *height = 0;
  *scale = 1;
  UIWindow* window = view.window;
  if (!window) {
    return;
  }
  UIWindowScene* scene = window.windowScene;
  if (!scene) {
    return;
  }
  CGRect bounds = scene.screen.bounds;
  *width = bounds.size.width;
  *height = bounds.size.height;
  *scale = scene.screen.nativeScale;
}

#cgo noescape displayInfo
#cgo nocallback displayInfo
static void displayInfo(float* width, float* height, float* scale, uintptr_t viewPtr) {
  *width = 0;
  *height = 0;
  *scale = 1;
  if (!viewPtr) {
    return;
  }
  UIView* view = (__bridge UIView*)(void*)viewPtr;
  if ([NSThread isMainThread]) {
    displayInfoOnMainThread(width, height, scale, view);
    return;
  }
  __block float w, h, s;
  dispatch_sync(dispatch_get_main_queue(), ^{
    displayInfoOnMainThread(&w, &h, &s, view);
  });
  *width = w;
  *height = h;
  *scale = s;
}

static void safeAreaOnMainThread(int *left, int *top, int *right, int *bottom, UIView* view) {
  *left = 0;
  *top = 0;
  *right = 0;
  *bottom = 0;

  UIWindow* window = view.window;
  if (!window) {
    return;
  }

  UIEdgeInsets insets = window.safeAreaInsets;
  *left = insets.left;
  *top = insets.top;
  *right = insets.right;
  *bottom = insets.bottom;
}

#cgo noescape safeArea
#cgo nocallback safeArea
static void safeArea(int *left, int *top, int *right, int *bottom, uintptr_t viewPtr) {
  *left = 0;
  *top = 0;
  *right = 0;
  *bottom = 0;
  if (!viewPtr) {
    return;
  }
  UIView* view = (__bridge UIView*)(void*)viewPtr;
  if ([NSThread isMainThread]) {
    safeAreaOnMainThread(left, top, right, bottom, view);
    return;
  }
  __block int l, t, r, b;
  dispatch_sync(dispatch_get_main_queue(), ^{
    safeAreaOnMainThread(&l, &t, &r, &b, view);
  });
  *left = l;
  *top = t;
  *right = r;
  *bottom = b;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"image"

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
		return o, GraphicsLibraryMetal, nil
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

func (u *UserInterface) SetUIView(uiview uintptr) error {
	u.uiView.Store(uiview)
	select {
	case err := <-u.errCh:
		return err
	case <-u.graphicsLibraryInitCh:
	}

	// This function should be called only when the graphics library is Metal.
	if g, ok := u.graphicsDriver.(interface{ SetUIView(uintptr) }); ok {
		g.SetUIView(uiview)
	}
	return nil
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

func (u *UserInterface) displayInfo() (int, int, float64, bool) {
	view := u.uiView.Load()
	if view == 0 {
		return 0, 0, 1, false
	}

	var cWidth, cHeight, cScale C.float
	C.displayInfo(&cWidth, &cHeight, &cScale, C.uintptr_t(view))
	scale := float64(cScale)
	width := int(dipFromNativePixels(float64(cWidth), scale))
	height := int(dipFromNativePixels(float64(cHeight), scale))
	return width, height, scale, true
}

func (u *UserInterface) safeArea() image.Rectangle {
	view := u.uiView.Load()
	if view == 0 {
		return image.Rectangle{}
	}

	var cWidth, cHeight, cScale C.float
	C.displayInfo(&cWidth, &cHeight, &cScale, C.uintptr_t(view))
	scale := float64(cScale)

	var cLeft, cTop, cRight, cBottom C.int
	C.safeArea(&cLeft, &cTop, &cRight, &cBottom, C.uintptr_t(view))

	x0 := int(dipFromNativePixels(float64(cLeft), scale))
	y0 := int(dipFromNativePixels(float64(cTop), scale))
	x1 := int(dipFromNativePixels(float64(cWidth)-float64(cRight), scale))
	y1 := int(dipFromNativePixels(float64(cHeight)-float64(cBottom), scale))
	return image.Rect(x0, y0, x1, y1)
}
