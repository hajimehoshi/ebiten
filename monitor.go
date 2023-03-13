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
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Monitor represents a monitor available to the system.
type Monitor = ui.Monitor

// WindowMonitor returns the current monitor. If a window has not been created, this will be nil.
func WindowMonitor() *Monitor {
	return ui.Get().Monitor()
}

// SetWindowMonitor sets the monitor that the window should be on. This can be called before or after Run.
func SetWindowMonitor(monitor *Monitor) {
	ui.Get().Window().SetMonitor(monitor.ID())
}

// SetWindowMonitorByID sets the monitor by its ID that the window should be on. This can be called before or after run.
func SetWindowMonitorByID(monitorID int) {
	ui.Get().Window().SetMonitor(monitorID)
}

// Monitors returns the monitors reported by the system.
func Monitors() (monitors []*Monitor) {
	return ui.Get().Monitors()
}
