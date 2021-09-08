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
	// GLFW only provides monitor x and y coordinates but no reliable sizes.
	// So the "correct" monitor is the one closest to x, y to the bottom right.
	// This usually works, but in some exceptional layouts may return the wrong monitor.
	// Thus, this function is best called with the top-left coordinates of an actual monitor if possible.
	var best *glfw.Monitor
	var bestScore int
	for _, mon := range glfw.GetMonitors() {
		mx, my := mon.GetPos()
		if x < mx || y < my {
			continue
		}
		score := (x - mx) + (y - my)
		if best == nil || score < bestScore {
			best, bestScore = mon, score
		}
	}
	if best == nil {
		return glfw.GetMonitors()[0]
	}
	return best
}
