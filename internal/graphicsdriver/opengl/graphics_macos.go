// Copyright 2024 The Ebitengine Authors
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

//go:build darwin && !ios

package opengl

import (
	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

type graphicsPlatform struct {
	presenter Presenter
}

// NewGraphics creates an implementation of graphicsdriver.Graphics for OpenGL.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	ctx, err := gl.NewDefaultContext()
	if err != nil {
		return nil, err
	}

	return newGraphics(ctx, color.ColorSpaceSRGB), nil
}

// SetPresenter sets what the rendered frame is presented through.
func (g *Graphics) SetPresenter(presenter Presenter) {
	g.presenter = presenter
}

func (g *Graphics) makeContextCurrent() error {
	return g.presenter.MakeContextCurrent()
}

func (g *Graphics) swapBuffers() error {
	// Call SwapInterval even though vsync is not changed.
	// When toggling to fullscreen, vsync state might be reset unexpectedly (#1787).

	// SwapInterval is affected by the current monitor of the window.
	// This needs to be called at least after SetMonitor.
	// Without SwapInterval after SetMonitor, vsync doesn't work (#375).
	var interval int
	if g.vsync {
		interval = 1
	}
	if err := g.presenter.SwapInterval(interval); err != nil {
		return err
	}

	if err := g.presenter.SwapBuffers(); err != nil {
		return err
	}
	return nil
}
