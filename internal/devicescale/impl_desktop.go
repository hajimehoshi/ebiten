// Copyright 2021 The Ebiten Authors
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

//go:build !android && !ios && !js
// +build !android,!ios,!js

package devicescale

import (
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

func monitorAt(x, y int) *glfw.Monitor {
	// Note: this assumes that x, y are exact monitor origins.
	// If they're not, this arbitrarily returns the first monitor.
	monitors := glfw.GetMonitors()
	for _, mon := range monitors {
		mx, my := mon.GetPos()
		if x == mx && y == my {
			return mon
		}
	}
	return monitors[0]
}

func impl(x, y int) float64 {
	// Keep calling GetContentScale until the returned scale is 0 (#2051).
	// Retry this at most 5 times to avoid an inifinite loop.
	for i := 0; i < 5; i++ {
		sx, _ := monitorAt(x, y).GetContentScale()
		if sx != 0 {
			return float64(sx)
		}
	}
	return 1
}
