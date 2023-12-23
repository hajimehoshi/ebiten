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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package opengl

import (
	"fmt"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
)

type graphicsPlatform struct {
	window *glfw.Window
}

// NewGraphics creates an implementation of graphicsdriver.Graphics for OpenGL.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	if microsoftgdk.IsXbox() {
		return nil, fmt.Errorf("opengl: OpenGL is not supported on Xbox")
	}

	ctx, err := gl.NewDefaultContext()
	if err != nil {
		return nil, err
	}

	if err := setGLFWClientAPI(ctx.IsES()); err != nil {
		return nil, err
	}

	return newGraphics(ctx), nil
}

func setGLFWClientAPI(isES bool) error {
	if isES {
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

func (g *Graphics) SetGLFWWindow(window *glfw.Window) {
	g.window = window
}

func (g *Graphics) makeContextCurrent() error {
	return g.window.MakeContextCurrent()
}

func (g *Graphics) swapBuffers() error {
	// Call SwapIntervals even though vsync is not changed.
	// When toggling to fullscreen, vsync state might be reset unexpectedly (#1787).

	// SwapInterval is affected by the current monitor of the window.
	// This needs to be called at least after SetMonitor.
	// Without SwapInterval after SetMonitor, vsynch doesn't work (#375).
	if g.vsync {
		if err := glfw.SwapInterval(1); err != nil {
			return err
		}
	} else {
		if err := glfw.SwapInterval(0); err != nil {
			return err
		}
	}

	if err := g.window.SwapBuffers(); err != nil {
		return err
	}
	return nil
}
