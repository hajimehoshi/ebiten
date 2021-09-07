// Copyright 2018 The Ebiten Authors
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

package devicescale

import (
	"sync"
)

type pos struct {
	x, y int
}

var (
	m                sync.Mutex
	cache            = map[pos]float64{}
	screenScaleCache = map[pos]float64{}
)

// GetAt returns the device scale at (x, y).
// x and y are in device-dependent pixels.
// The device scale maps device dependent pixels to device independent pixels.
func GetAt(x, y int) float64 {
	m.Lock()
	defer m.Unlock()
	if s, ok := cache[pos{x, y}]; ok {
		return s
	}
	s := impl(x, y)
	cache[pos{x, y}] = s

	// TODO: Provide a way to invalidate the cache, or move the cache.
	// The device scale can vary even for the same monitor.
	// The only known case is when the application works on macOS, with OpenGL, with a wider screen mode,
	// and in the fullscreen mode (#1573).

	return s
}

// ScreenScaleAt returns the screen scale at (x, y).
// The screen scale maps physical screen pixels to device dependent pixels.
func ScreenScaleAt(x, y int) float64 {
	m.Lock()
	defer m.Unlock()
	if s, ok := screenScaleCache[pos{x, y}]; ok {
		return s
	}
	s := screenScaleImpl(x, y)
	screenScaleCache[pos{x, y}] = s
	return s
}
