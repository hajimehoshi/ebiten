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

package ebiten

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// MonitorID represents a monitor's identifier.
type MonitorID int

// Monitor represents a monitor available to the system.
type Monitor struct {
	id            MonitorID
	name          string
	x, y          int
	width, height int
}

// Bounds returns the position and size of the monitor.
func (m *Monitor) Bounds() image.Rectangle {
	return image.Rect(m.x, m.y, m.x+m.width, m.y+m.height)
}

// ID returns the monitor's ID. 0 is always the primary monitor.
func (m *Monitor) ID() MonitorID {
	return m.id
}

// Name returns the monitor's name. On Linux, this reports the monitors in xrandr format. On Windows, this reports "Generic PnP Monitor" for all monitors.
func (m *Monitor) Name() string {
	return m.name
}

func uiMonitorToMonitor(m *ui.Monitor) (monitor Monitor) {
	monitor.x = m.Bounds().Min.X
	monitor.y = m.Bounds().Min.Y
	monitor.width = m.Bounds().Dx()
	monitor.height = m.Bounds().Dy()
	monitor.id = MonitorID(m.ID())
	monitor.name = m.Name()
	return
}

// WindowMonitor returns the current monitor.
func WindowMonitor() Monitor {
	m := ui.Get().Monitor()
	if m == nil {
		return Monitor{}
	}
	return uiMonitorToMonitor(m)
}

// SetWindowMonitor sets the monitor that the window should be on. This can be called before or after Run.
func SetWindowMonitor(monitor Monitor) {
	ui.Get().Window().SetMonitor(int(monitor.id))
}

// AppendMonitors returns the monitors reported by the system.
func AppendMonitors(monitors []Monitor) []Monitor {
	for _, m := range ui.Get().Monitors() {
		monitors = append(monitors, uiMonitorToMonitor(m))
	}
	return monitors
}
