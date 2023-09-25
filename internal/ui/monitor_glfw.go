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

//go:build !android && !ios && !js && !nintendosdk

package ui

import (
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// Monitor is a wrapper around glfw.Monitor.
type Monitor struct {
	m         *glfw.Monitor
	videoMode *glfw.VidMode

	id              int
	name            string
	x               int
	y               int
	contentScale    float64
	videoModeScale_ float64
}

// Name returns the monitor's name.
func (m *Monitor) Name() string {
	return m.name
}

func (m *Monitor) deviceScaleFactor() float64 {
	// It is rare, but monitor can be nil when glfw.GetPrimaryMonitor returns nil.
	// In this case, return 1 as a tentative scale (#1878).
	if m == nil {
		return 1
	}
	return m.contentScale
}

func (m *Monitor) videoModeScale() float64 {
	// It is rare, but monitor can be nil when glfw.GetPrimaryMonitor returns nil.
	// In this case, return 1 as a tentative scale (#1878).
	if m == nil {
		return 1
	}
	return m.videoModeScale_
}

type monitors struct {
	// monitors is the monitor list cache for desktop glfw compile targets.
	// populated by 'updateMonitors' which is called on init and every
	// monitor config change event.
	monitors []*Monitor

	m sync.Mutex

	updateCalled int32
}

var theMonitors monitors

func (m *monitors) append(ms []*Monitor) []*Monitor {
	if atomic.LoadInt32(&m.updateCalled) == 0 {
		panic("ui: (*monitors).update must be called before (*monitors).append is called")
	}

	m.m.Lock()
	defer m.m.Unlock()

	return append(ms, m.monitors...)
}

func (m *monitors) monitorFromGLFWMonitor(glfwMonitor *glfw.Monitor) *Monitor {
	m.m.Lock()
	defer m.m.Unlock()

	for _, m := range m.monitors {
		if m.m == glfwMonitor {
			return m
		}
	}
	return nil
}

func (m *monitors) monitorFromID(id int) *Monitor {
	m.m.Lock()
	defer m.m.Unlock()

	return m.monitors[id]
}

// monitorFromPosition returns a monitor for the given position (x, y),
// or returns nil if monitor is not found.
func (m *monitors) monitorFromPosition(x, y int) *Monitor {
	m.m.Lock()
	defer m.m.Unlock()

	for _, m := range m.monitors {
		// TODO: Fix incorrectness in the cases of https://github.com/glfw/glfw/issues/1961.
		if m.x <= x && x < m.x+m.videoMode.Width && m.y <= y && y < m.y+m.videoMode.Height {
			return m
		}
	}
	return nil
}

// update must be called from the main thread.
func (m *monitors) update() {
	glfwMonitors := glfw.GetMonitors()
	newMonitors := make([]*Monitor, 0, len(glfwMonitors))
	for i, m := range glfwMonitors {
		x, y := m.GetPos()

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

		newMonitors = append(newMonitors, &Monitor{
			m:               m,
			videoMode:       m.GetVideoMode(),
			id:              i,
			name:            m.GetName(),
			x:               x,
			y:               y,
			contentScale:    contentScale,
			videoModeScale_: videoModeScale(m),
		})
	}

	m.m.Lock()
	m.monitors = newMonitors
	m.m.Unlock()

	atomic.StoreInt32(&m.updateCalled, 1)
}
