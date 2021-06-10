// Copyright 2020 The Ebiten Authors
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

//go:build (freebsd || linux) && !android
// +build freebsd linux
// +build !android

package opengl

import (
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

var (
	pboAvailable     = true
	pboAvailableOnce sync.Once
)

func isPBOAvailable() bool {
	pboAvailableOnce.Do(func() {
		// This must be called after GL context is created.
		str := gl.RendererDeviceString()
		tokens := strings.Split(str, " ")
		if len(tokens) == 0 {
			return
		}
		// Raspbery Pi 4 has an issue around PBO (#1208).
		if tokens[0] == "V3D" {
			pboAvailable = false
		}
	})
	return pboAvailable
}
