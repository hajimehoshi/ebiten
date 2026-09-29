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

//go:build linux && (amd64 || arm64) && !android && !nintendosdk && !playstation5

package ui

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2/internal/gbm"
)

// maybeNewGBMBackend probes EGL and KMS before choosing the backend without a window system.
// An unusable DRM device can then fall back to fbdev before the game starts.
func maybeNewGBMBackend(u *UserInterface) (uiBackend, error) {
	display, err := gbm.OpenDisplay()
	if err != nil {
		return nil, err
	}
	c, err := gbm.NewContext(display)
	if err != nil {
		return nil, errors.Join(err, display.Close())
	}

	err = c.Probe()
	if err != nil {
		err = errors.Join(err, c.Close())
	}
	if err != nil {
		return nil, errors.Join(err, display.Close())
	}

	width, height := c.Size()
	return newNoWindowSystemBackend(u, width, height, c, nil, display.Close), nil
}
