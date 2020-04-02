// Copyright 2020 The Ebiten Authors
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

// +build js

package monogame

import (
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

type Graphics struct {
	t *thread.Thread
}

var theGraphics Graphics

func Get() *Graphics {
	return &theGraphics
}

func (g *Graphics) SetThread(thread *thread.Thread) {
	g.t = thread
}

func (g *Graphics) Begin() {
}

func (g *Graphics) End() {
}

func (g *Graphics) SetTransparent(transparent bool) {
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) {
}

func (g *Graphics) NewImage(width, height int) (driver.Image, error) {
	return &Image{
		width:  width,
		height: height,
	}, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (driver.Image, error) {
	return &Image{
		width:  width,
		height: height,
	}, nil
}

func (g *Graphics) Reset() error {
	return nil
}

func (g *Graphics) Draw(indexLen int, indexOffset int, mode driver.CompositeMode, colorM *affine.ColorM, filter driver.Filter, address driver.Address) error {
	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	// TODO: Implement this
}

func (g *Graphics) VDirection() driver.VDirection {
	return driver.VDownward
}

func (g *Graphics) NeedsRestoring() bool {
	return false
}

func (g *Graphics) IsGL() bool {
	return false
}

func (g *Graphics) HasHighPrecisionFloat() bool {
	return true
}

func (g *Graphics) MaxImageSize() int {
	// TODO: Implement this
	return 4096
}
