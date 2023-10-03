// Copyright 2022 The Ebitengine Authors
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

//go:build !android && !ios && !js && !nintendosdk

package opengl

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

func (g *Graphics) SetGLFWClientAPI() error {
	if g.context.ctx.IsES() {
		if err := glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI); err != nil {
			return err
		}
		if err := glfw.WindowHint(glfw.ContextVersionMajor, 3); err != nil {
			return err
		}
		if err := glfw.WindowHint(glfw.ContextVersionMinor, 0); err != nil {
			return err
		}
		if err := glfw.WindowHint(glfw.ContextCreationAPI, glfw.EGLContextAPI); err != nil {
			return err
		}
		return nil
	}

	if err := glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI); err != nil {
		return err
	}
	if err := glfw.WindowHint(glfw.ContextVersionMajor, 3); err != nil {
		return err
	}
	if err := glfw.WindowHint(glfw.ContextVersionMinor, 2); err != nil {
		return err
	}
	// macOS requires forward-compatible and a core profile.
	if runtime.GOOS == "darwin" {
		if err := glfw.WindowHint(glfw.OpenGLForwardCompat, glfw.True); err != nil {
			return err
		}
		if err := glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile); err != nil {
			return err
		}
	}
	return nil
}
