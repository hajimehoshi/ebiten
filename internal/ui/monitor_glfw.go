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

	"github.com/hajimehoshi/ebiten/v2/internal/devicescale"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// Monitor is a wrapper around glfw.Monitor.
type Monitor struct {
	m         *glfw.Monitor
	videoMode *glfw.VidMode

	id             int
	name           string
	x              int
	y              int
	videoModeScale float64
}

// Name returns the monitor's name.
func (m *Monitor) Name() string {
	return m.name
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

// update must be called from the main thread.
func (m *monitors) update() {
	glfwMonitors := glfw.GetMonitors()
	newMonitors := make([]*Monitor, 0, len(glfwMonitors))
	for i, m := range glfwMonitors {
		x, y := m.GetPos()
		newMonitors = append(newMonitors, &Monitor{
			m:              m,
			videoMode:      m.GetVideoMode(),
			id:             i,
			name:           m.GetName(),
			x:              x,
			y:              y,
			videoModeScale: videoModeScale(m),
		})
	}

	m.m.Lock()
	m.monitors = newMonitors
	m.m.Unlock()

	devicescale.ClearCache()

	atomic.StoreInt32(&m.updateCalled, 1)
}
