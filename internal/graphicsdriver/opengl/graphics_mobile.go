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

//go:build android || ios
// +build android ios

package opengl

import (
	"golang.org/x/mobile/gl"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gles"
)

func init() {
	theGraphics.context.ctx = gles.DefaultContext{}
}

func (g *Graphics) SetGomobileGLContext(context gl.Context) {
	g.context.ctx = gles.NewGomobileContext(context)
}
