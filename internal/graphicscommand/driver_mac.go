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

// +build darwin,!ios
// +build !js

package graphicscommand

import (
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/opengl"
)

// isMetalSupported represents whether Metal is supported.
var isMetalSupported = true

func init() {
	if _, err := mtl.CreateSystemDefaultDevice(); err != nil {
		isMetalSupported = false
	}
}

func Driver() graphicsdriver.GraphicsDriver {
	if isMetalSupported {
		return metal.Get()
	} else {
		return opengl.Get()
	}
}
