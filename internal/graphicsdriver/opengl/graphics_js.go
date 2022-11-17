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
	"os"
	"strings"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

// NewGraphics creates an implementation of graphicsdriver.Graphics for OpenGL.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics(canvas js.Value) (graphicsdriver.Graphics, error) {
	var glContext js.Value

	attr := js.Global().Get("Object").New()
	attr.Set("alpha", true)
	attr.Set("premultipliedAlpha", true)
	attr.Set("stencil", true)

	var webGL2 bool
	if webGL2MightBeAvailable() {
		glContext = canvas.Call("getContext", "webgl2", attr)
	}

	if glContext.Truthy() {
		webGL2 = true
	} else {
		// Even though WebGL2RenderingContext exists, getting a webgl2 context might fail (#1738).
		glContext = canvas.Call("getContext", "webgl", attr)
		if !glContext.Truthy() {
			glContext = canvas.Call("getContext", "experimental-webgl", attr)
		}
	}

	if !glContext.Truthy() {
		return nil, fmt.Errorf("opengl: getContext failed")
	}

	ctx, err := gl.NewDefaultContext(glContext)
	if err != nil {
		return nil, err
	}

	g := &Graphics{}
	g.context.canvas = canvas
	g.context.ctx = ctx
	g.context.webGL2 = webGL2

	if !webGL2 {
		glContext.Call("getExtension", "OES_standard_derivatives")
	}

	return g, nil
}

func webGL2MightBeAvailable() bool {
	env := os.Getenv("EBITENGINE_OPENGL")
	for _, t := range strings.Split(env, ",") {
		switch strings.TrimSpace(t) {
		case "webgl1":
			return false
		}
	}
	return js.Global().Get("WebGL2RenderingContext").Truthy()
}
