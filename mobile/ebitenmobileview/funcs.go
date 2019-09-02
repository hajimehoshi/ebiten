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

// Package ebitenmobileview offers functions for OpenGL/Metal view of mobiles.
//
// The functions are not intended for public usages.
// There is no guarantee of backward compatibility.
package ebitenmobileview

import (
	"math"
	"runtime"
)

type ViewRectSetter interface {
	SetViewRect(x, y, width, height int)
}

func Layout(viewWidth, viewHeight int, viewRectSetter ViewRectSetter) {
	theState.m.Lock()
	defer theState.m.Unlock()
	layout(viewWidth, viewHeight, viewRectSetter)
}

func layout(viewWidth, viewHeight int, viewRectSetter ViewRectSetter) {
	if theState.game == nil {
		// It is fine to override the existing function since only the last layout result matters.
		theState.delayedLayout = func() {
			layout(viewWidth, viewHeight, viewRectSetter)
		}
		return
	}

	w, h := theState.game.Layout(int(viewWidth), int(viewHeight))
	scaleX := float64(viewWidth) / float64(w)
	scaleY := float64(viewHeight) / float64(h)
	scale := math.Min(scaleX, scaleY)

	width := int(math.Ceil(float64(w) * scale))
	height := int(math.Ceil(float64(h) * scale))
	x := (viewWidth - width) / 2
	y := (viewHeight - height) / 2

	if !theState.running {
		start(theState.game.Update, w, h, scale)
		theState.running = true
	} else {
		setScreenSize(w, h, scale)
	}

	if viewRectSetter != nil {
		viewRectSetter.SetViewRect(x, y, width, height)
	}
}

func Update() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	theState.m.Lock()
	defer theState.m.Unlock()

	return update()
}

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	updateTouchesOnAndroid(action, id, x, y)
}

func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	updateTouchesOnIOSImpl(phase, ptr, x, y)
}

func SetUIView(uiview int64) {
	setUIView(uintptr(uiview))
}
