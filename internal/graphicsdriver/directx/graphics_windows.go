// Copyright 2022 The Ebiten Authors
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

package directx

import (
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// NewGraphics creates an implementation of graphicsdriver.Graphics for DirectX.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	var (
		useWARP       bool
		useDebugLayer bool
	)
	env := os.Getenv("EBITENGINE_DIRECTX")
	if env == "" {
		// For backward compatibility, read the EBITEN_ version.
		env = os.Getenv("EBITEN_DIRECTX")
	}
	for _, t := range strings.Split(env, ",") {
		switch strings.TrimSpace(t) {
		case "warp":
			// TODO: Is WARP available on Xbox?
			useWARP = true
		case "debug":
			useDebugLayer = true
		}
	}

	// Specify the level 11 by default.
	// Some old cards don't work well with the default feature level (#2447, #2486).
	var featureLevel _D3D_FEATURE_LEVEL = _D3D_FEATURE_LEVEL_11_0
	if env := os.Getenv("EBITENGINE_DIRECTX_FEATURE_LEVEL"); env != "" {
		switch env {
		case "11_0":
			featureLevel = _D3D_FEATURE_LEVEL_11_0
		case "11_1":
			featureLevel = _D3D_FEATURE_LEVEL_11_1
		case "12_0":
			featureLevel = _D3D_FEATURE_LEVEL_12_0
		case "12_1":
			featureLevel = _D3D_FEATURE_LEVEL_12_1
		case "12_2":
			featureLevel = _D3D_FEATURE_LEVEL_12_2
		}
	}

	// TODO: Implement a new graphics for DirectX 11 (#2613).
	g, err := newGraphics12(useWARP, useDebugLayer, featureLevel)
	if err != nil {
		return nil, err
	}
	return g, nil
}
