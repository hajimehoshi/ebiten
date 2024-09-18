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

package opengl

import (
	"fmt"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

type graphicsPlatform struct {
}

// NewGraphics creates an implementation of graphicsdriver.Graphics for OpenGL.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics(canvas js.Value, colorSpace graphicsdriver.ColorSpace) (graphicsdriver.Graphics, error) {
	var glContext js.Value

	attr := js.Global().Get("Object").New()
	attr.Set("alpha", true)
	attr.Set("premultipliedAlpha", true)
	attr.Set("stencil", true)

	glContext = canvas.Call("getContext", "webgl2", attr)

	if !glContext.Truthy() {
		return nil, fmt.Errorf("opengl: getContext for webgl2 failed")
	}

	switch colorSpace {
	case graphicsdriver.ColorSpaceSRGB:
		glContext.Set("drawingBufferColorSpace", "srgb")
	case graphicsdriver.ColorSpaceDisplayP3:
		glContext.Set("drawingBufferColorSpace", "display-p3")
	}

	ctx, err := gl.NewDefaultContext(glContext)
	if err != nil {
		return nil, err
	}

	return newGraphics(ctx), nil
}

func (g *Graphics) makeContextCurrent() error {
	return nil
}

func (g *Graphics) swapBuffers() error {
	return nil
}
