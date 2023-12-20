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

var (
	theEGL egl
)

func NewGraphics(nativeWindowType uintptr) (graphicsdriver.Graphics, error) {
	theEGL.init(nativeWindowType)

	ctx, err := gl.NewDefaultContext()
	if err != nil {
		return nil, err
	}
	return newGraphics(ctx), nil
}

func (g *Graphics) makeContextCurrent() {
	theEGL.makeContextCurrent()
}

func (g *Graphics) swapBuffers() error {
	theEGL.swapBuffers()
	return nil
}
