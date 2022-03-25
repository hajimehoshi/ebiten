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

//go:build ios
// +build ios

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverGetterImpl struct {
	gomobileBuild bool
}

func (g *graphicsDriverGetterImpl) getAuto() graphicsdriver.Graphics {
	if m := g.getMetal(); m != nil {
		return m
	}
	return g.getOpenGL()
}

func (*graphicsDriverGetterImpl) getOpenGL() graphicsdriver.Graphics {
	if g := opengl.Get(); g != nil {
		return g
	}
	return nil
}

func (*graphicsDriverGetterImpl) getDirectX() graphicsdriver.Graphics {
	return nil
}

func (g *graphicsDriverGetterImpl) getMetal() graphicsdriver.Graphics {
	// When gomobile-build is used, GL functions must be called via
	// gl.Context so that they are called on the appropriate thread.
	if g.gomobileBuild {
		return nil
	}
	if m := metal.Get(); m != nil {
		return m
	}
	return nil
}

func SetUIView(uiview uintptr) {
	// This function should be called only when the graphics library is Metal.
	if g, ok := theUI.graphicsDriver.(interface{ SetUIView(uintptr) }); ok {
		g.SetUIView(uiview)
	}
}

func IsGL() bool {
	return theUI.graphicsDriver.IsGL()
}
