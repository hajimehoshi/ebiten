// Copyright 2023 The Ebitengine Authors
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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"image"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// Monitor is a wrapper around glfw.Monitor.
type Monitor struct {
	m         *glfw.Monitor
	videoMode *glfw.VidMode

	id                 int
	name               string
	boundsInGLFWPixels image.Rectangle
	contentScale       float64
}

// Name returns the monitor's name.
func (m *Monitor) Name() string {
	return m.name
}

// DeviceScaleFactor is concurrent-safe as contentScale is immutable.
func (m *Monitor) DeviceScaleFactor() float64 {
	return m.contentScale
}

// Size returns the size of the monitor in device-independent pixels.
func (m *Monitor) Size() (int, int) {
	w, h := m.sizeInDIP()
	return int(w), int(h)
}

func (m *Monitor) sizeInDIP() (float64, float64) {
	w, h := m.boundsInGLFWPixels.Dx(), m.boundsInGLFWPixels.Dy()
	s := m.DeviceScaleFactor()
	return dipFromGLFWPixel(float64(w), s), dipFromGLFWPixel(float64(h), s)
}

type monitors struct {
	// monitors is the monitor list cache for desktop glfw compile targets.
	// populated by 'updateMonitors' which is called on init and every
	// monitor config change event.
	monitors []*Monitor

	m sync.Mutex

	updateCalled atomic.Bool
}

var theMonitors monitors

func (m *monitors) append(ms []*Monitor) []*Monitor {
	if !m.updateCalled.Load() {
		panic("ui: (*monitors).update must be called before (*monitors).append is called")
	}

	m.m.Lock()
	defer m.m.Unlock()

	return append(ms, m.monitors...)
}

func (m *monitors) primaryMonitor() *Monitor {
	if !m.updateCalled.Load() {
		panic("ui: (*monitors).update must be called before (*monitors).primaryMonitor is called")
	}

	m.m.Lock()
	defer m.m.Unlock()

	// GetMonitors might return nil in theory (#1878, #1887).
	// primaryMonitor can be called at the initialization, so monitors can be nil.
	if len(m.monitors) == 0 {
		return nil
	}
	return m.monitors[0]
}

// monitorFromPosition returns a monitor for the given position (x, y),
// or returns nil if monitor is not found.
// The position is in GLFW pixels.
func (m *monitors) monitorFromPosition(x, y int) *Monitor {
	m.m.Lock()
	defer m.m.Unlock()

	for _, m := range m.monitors {
		// Use an inclusive range. On macOS, the cursor position can take this range (#2794).
		b := m.boundsInGLFWPixels
		if b.Min.X <= x && x <= b.Max.X && b.Min.Y <= y && y <= b.Max.Y {
			return m
		}
	}
	return nil
}

// update must be called from the main thread.
func (m *monitors) update() error {
	glfwMonitors, err := glfw.GetMonitors()
	if err != nil {
		return err
	}
	newMonitors := make([]*Monitor, 0, len(glfwMonitors))
	for i, m := range glfwMonitors {
		x, y, err := m.GetPos()
		if err != nil {
			return err
		}

		// TODO: Detect the update of the content scale by SetContentScaleCallback (#2343).
		contentScale := 1.0

		// Keep calling GetContentScale until the returned scale is 0 (#2051).
		// Retry this at most 5 times to avoid an infinite loop.
		for i := 0; i < 5; i++ {
			// An error can happen e.g. when entering a screensaver on Windows (#2488).
			sx, _, err := m.GetContentScale()
			if err != nil {
				continue
			}
			if sx == 0 {
				continue
			}
			contentScale = float64(sx)
			break
		}

		videoMode, err := m.GetVideoMode()
		if err != nil {
			return err
		}
		name, err := m.GetName()
		if err != nil {
			return err
		}
		w, h, err := glfwMonitorSizeInGLFWPixels(m)
		if err != nil {
			return err
		}
		b := image.Rect(x, y, x+w, y+h)
		newMonitors = append(newMonitors, &Monitor{
			m:                  m,
			videoMode:          videoMode,
			id:                 i,
			name:               name,
			boundsInGLFWPixels: b,
			contentScale:       contentScale,
		})
	}

	m.m.Lock()
	m.monitors = newMonitors
	m.m.Unlock()

	m.updateCalled.Store(true)
	return nil
}
