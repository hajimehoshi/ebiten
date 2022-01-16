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
	m     sync.Mutex
	cache = map[pos]float64{}
)

// GetAt returns the device scale at (x, y), i.e. the number of device-dependent pixels per device-independent pixel.
// x and y are in device-dependent pixels and must be the top-left coordinate of a monitor, or 0,0 to request a "global scale".
func GetAt(x, y int) float64 {
	m.Lock()
	defer m.Unlock()
	if s, ok := cache[pos{x, y}]; ok {
		return s
	}
	s := impl(x, y)
	cache[pos{x, y}] = s
	return s
}

// CelarCache clears the cache.
// This should be called when monitors are changed by connecting or disconnecting.
func ClearCache() {
	// TODO: This should be called not only when monitors are changed but also a monitor's scales are changed.
	// The device scale can vary even for the same monitor.
	// The only known case is when the application works on macOS, with OpenGL, with a wider screen mode,
	// and in the fullscreen mode (#1573).

	m.Lock()
	defer m.Unlock()
	for k := range cache {
		delete(cache, k)
	}
}
