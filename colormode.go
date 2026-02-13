// Copyright 2026 The Ebitengine Authors
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
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
)

// ColorMode represents a color scheme, such as light or dark.
type ColorMode int

const (
	// ColorModeUnknown represents an unknown color mode.
	ColorModeUnknown ColorMode = ColorMode(colormode.Unknown)

	// ColorModeLight represents the light color mode.
	ColorModeLight ColorMode = ColorMode(colormode.Light)

	// ColorModeDark is the dark color mode.
	ColorModeDark ColorMode = ColorMode(colormode.Dark)
)

// SystemColorMode returns the system color mode.
//
// If the current environment doesn't support this feature, SystemColorMode returns ColorModeUnknown.
//
// SystemColorMode is concurrent-safe.
func SystemColorMode() ColorMode {
	return theSystemColorCache.get()
}

var theSystemColorCache systemColorCache

type systemColorCache struct {
	mode        atomic.Int32
	lastUpdated atomic.Pointer[time.Time]
	m           sync.Mutex
}

func (s *systemColorCache) get() ColorMode {
	if t := s.lastUpdated.Load(); t != nil && time.Since(*t) < time.Second {
		return ColorMode(s.mode.Load())
	}

	s.m.Lock()
	defer s.m.Unlock()

	now := time.Now()
	if t := s.lastUpdated.Load(); t != nil && now.Sub(*t) < time.Second {
		return ColorMode(s.mode.Load())
	}

	clr := colormode.SystemColorMode()
	s.mode.Store(int32(clr))
	s.lastUpdated.Store(&now)
	return ColorMode(clr)
}
