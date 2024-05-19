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

//go:build playstation5

package playstation5

// #include "graphics_playstation5.h"
import "C"

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type playstation5Error struct {
	name    string
	code    int
	message string
}

func newPlaystation5Error(name string, err C.ebitengine_Error) *playstation5Error {
	return &playstation5Error{
		name:    name,
		code:    int(err.code),
		message: C.GoString(err.message),
	}
}

func (e *playstation5Error) Error() string {
	return fmt.Sprintf("playstation5: error at %s, code: %d, message: %s", e.name, e.code, e.message)
}

type Graphics struct {
}

func NewGraphics() (*Graphics, error) {
	return &Graphics{}, nil
}

func (g *Graphics) Initialize() error {
	if err := C.ebitengine_InitializeGraphics(); !C.ebitengine_IsErrorNil(&err) {
		return newPlaystation5Error("(*playstation5.Graphics).Initialize", err)
	}
	return nil
}

func (g *Graphics) Begin() error {
	return nil
}

func (g *Graphics) End(present bool) error {
	return nil
}

func (g *Graphics) SetTransparent(transparent bool) {
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint32) error {
	return nil
}

func (g *Graphics) NewImage(width, height int) (graphicsdriver.Image, error) {
	var id C.int
	if err := C.ebitengine_NewImage(&id, C.int(width), C.int(height)); !C.ebitengine_IsErrorNil(&err) {
		return nil, newPlaystation5Error("(*playstation5.Graphics).NewImage", err)
	}
	return &Image{
		id: graphicsdriver.ImageID(id),
	}, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	var id C.int
	if err := C.ebitengine_NewScreenFramebufferImage(&id, C.int(width), C.int(height)); !C.ebitengine_IsErrorNil(&err) {
		return nil, newPlaystation5Error("(*playstation5.Graphics).NewScreenFramebufferImage", err)
	}
	return &Image{
		id: graphicsdriver.ImageID(id),
	}, nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
}

func (g *Graphics) NeedsClearingScreen() bool {
	return false
}

func (g *Graphics) MaxImageSize() int {
	return 4096 // TODO: Get the value from the SDK.
}

func (g *Graphics) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	var id C.int
	// TODO: Give a source code.
	if err := C.ebitengine_NewShader(&id, nil); !C.ebitengine_IsErrorNil(&err) {
		return nil, newPlaystation5Error("(*playstation5.Graphics).NewShader", err)
	}
	return &Shader{
		id: graphicsdriver.ShaderID(id),
	}, nil
}

func (g *Graphics) DrawTriangles(dst graphicsdriver.ImageID, srcs [graphics.ShaderImageCount]graphicsdriver.ImageID, shader graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32, fillRule graphicsdriver.FillRule) error {
	return nil
}

type Image struct {
	id graphicsdriver.ImageID
}

func (i *Image) ID() graphicsdriver.ImageID {
	return i.id
}

func (i *Image) Dispose() {
	C.ebitengine_DisposeImage(C.int(i.id))
}

func (i *Image) ReadPixels(args []graphicsdriver.PixelsArgs) error {
	// TODO: Implement this
	return nil
}

func (i *Image) WritePixels(args []graphicsdriver.PixelsArgs) error {
	// TODO: Implement this
	return nil
}

type Shader struct {
	id graphicsdriver.ShaderID
}

func (s *Shader) ID() graphicsdriver.ShaderID {
	return s.id
}

func (s *Shader) Dispose() {
	C.ebitengine_DisposeShader(C.int(s.id))
}
