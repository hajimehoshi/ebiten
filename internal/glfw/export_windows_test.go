// Copyright 2026 The Ebitengine Authors
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

package glfw

import (
	"runtime"
	"time"
	"unsafe"
)

func PackPoint64(x, y int32) uint64 {
	return (_POINT{x: x, y: y}).pack64()
}

const WheelPageScroll = _WHEEL_PAGESCROLL

func WheelScrollAmount(notches float64, setting uint32) (float64, ScrollUnit) {
	return wheelScrollAmount(notches, setting)
}

var SetFocusForTesting = _SetFocus

type DirectManipulationGesture struct {
	g directManipulationGesture
}

func NewDirectManipulationGesture() *DirectManipulationGesture {
	return &DirectManipulationGesture{
		g: newDirectManipulationGesture(),
	}
}

func (d *DirectManipulationGesture) Transform(m [6]float32) (panX, panY, pinch float64) {
	return d.g.transform(m)
}

const (
	DirectManipulationBuilding = _DIRECTMANIPULATION_BUILDING
	DirectManipulationRunning  = _DIRECTMANIPULATION_RUNNING
	DirectManipulationInertia  = _DIRECTMANIPULATION_INERTIA
	DirectManipulationReady    = _DIRECTMANIPULATION_READY

	DirectManipulationContactPollingDuration = directManipulationContactPollingDuration
)

// DirectManipulationPolling reports whether a viewport has to be polled soon after its status changed as
// statuses. If contact is true, a contact was handed to the viewport sinceContact ago, before the statuses.
func DirectManipulationPolling(contact bool, sinceContact time.Duration, statuses ...uint32) bool {
	now := time.Now()
	d := &directManipulation{}
	if contact {
		d.contactTime = now.Add(-sinceContact)
	}
	for _, s := range statuses {
		d.setStatus(s)
	}
	return d.polling(now)
}

// DirectManipulationVtblSlots returns the number of methods in each vtable, which must match
// directmanipulation.idl.
func DirectManipulationVtblSlots() map[string]uintptr {
	ptr := unsafe.Sizeof(uintptr(0))
	return map[string]uintptr{
		"IDirectManipulationManager":       unsafe.Sizeof(_IDirectManipulationManager_Vtbl{}) / ptr,
		"IDirectManipulationUpdateManager": unsafe.Sizeof(_IDirectManipulationUpdateManager_Vtbl{}) / ptr,
		"IDirectManipulationViewport":      unsafe.Sizeof(_IDirectManipulationViewport_Vtbl{}) / ptr,
		"IDirectManipulationContent":       unsafe.Sizeof(_IDirectManipulationContent_Vtbl{}) / ptr,
	}
}

// DirectManipulationZoomToRect zooms the viewport of Direct Manipulation for a hidden window to the given
// rectangle, and returns the content transform. It returns nil without an error when Direct Manipulation is
// not available.
func DirectManipulationZoomToRect(left, top, right, bottom float32) (*[6]float32, error) {
	// Direct Manipulation is used from the thread that owns the window.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hwnd, err := _CreateWindowExW(0, "STATIC", "", 0, 0, 0, directManipulationViewportSize, directManipulationViewportSize, 0, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = _DestroyWindow(hwnd)
	}()

	d, err := newDirectManipulation(hwnd)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}
	defer d.release(hwnd)

	if err := d.viewport.ZoomToRect(d.zoomToRect, left, top, right, bottom, false); err != nil {
		return nil, err
	}
	if err := d.updateManager.Update(); err != nil {
		return nil, err
	}
	m, err := d.content.GetContentTransform()
	if err != nil {
		return nil, err
	}
	return &m, nil
}
