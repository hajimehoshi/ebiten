// Copyright 2023 The Ebitengine Authors
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

//go:build nintendosdk

package opengl

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

type graphicsPlatform struct {
	egl *egl
}

func NewGraphics(nativeWindowType uintptr) (graphicsdriver.Graphics, error) {
	ctx, err := gl.NewDefaultContext()
	if err != nil {
		return nil, err
	}
	g := newGraphics(ctx)
	e, err := newEGL(nativeWindowType)
	if err != nil {
		return nil, err
	}
	g.egl = e
	return g, nil
}

func (g *Graphics) makeContextCurrent() error {
	return g.egl.makeContextCurrent()
}

func (g *Graphics) swapBuffers() error {
	g.egl.swapBuffers()
	return nil
}
