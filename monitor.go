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

// Monitor represents a monitor available to the system.
type Monitor ui.Monitor

// Bounds returns the position and size of the monitor.
func (m *Monitor) Bounds() image.Rectangle {
	return (*ui.Monitor)(m).Bounds()
}

// Name returns the monitor's name. On Linux, this reports the monitors in xrandr format. On Windows, this reports "Generic PnP Monitor" for all monitors.
func (m *Monitor) Name() string {
	return (*ui.Monitor)(m).Name()
}

// WindowMonitor returns the current monitor.
func WindowMonitor() *Monitor {
	m := ui.Get().Monitor()
	if m == nil {
		return nil
	}
	return (*Monitor)(m)
}

// SetWindowMonitor sets the monitor that the window should be on. This can be called before or after Run.
func SetWindowMonitor(monitor *Monitor) {
	ui.Get().Window().SetMonitor((*ui.Monitor)(monitor))
}

// AppendMonitors returns the monitors reported by the system. On desktop platforms, there will always be at least one monitor appended and the first monitor in the slice will be the primary monitor. Any monitors added or removed will show up with subsequent calls to this function.
func AppendMonitors(monitors []*Monitor) []*Monitor {
	for _, m := range ui.Get().AppendMonitors(nil) {
		monitors = append(monitors, (*Monitor)(m))
	}
	return monitors
}
