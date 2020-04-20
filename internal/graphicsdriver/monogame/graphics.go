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
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/monogame"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

type Graphics struct {
	game *monogame.Game
}

var theGraphics Graphics

func Get() *Graphics {
	return &theGraphics
}

func (g *Graphics) SetGame(game *monogame.Game) {
	g.game = game
}

func (g *Graphics) SetThread(thread *thread.Thread) {
	panic("monogame: SetThread is not implemented")
}

func (g *Graphics) Begin() {
	// Do nothing
}

func (g *Graphics) End() {
	// Do nothing
}

func (g *Graphics) SetTransparent(transparent bool) {
	panic("monogame: SetTransparent is not implemented yet")
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) {
	g.game.SetVertices(vertices, indices)
}

func (g *Graphics) NewImage(width, height int) (driver.Image, error) {
	w, h := graphics.InternalImageSize(width), graphics.InternalImageSize(height)
	v := g.game.NewRenderTarget2D(w, h)
	return &Image{
		v:      v,
		g:      g,
		width:  width,
		height: height,
	}, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (driver.Image, error) {
	return &Image{
		v:      &screen{game: g.game},
		g:      g,
		width:  width,
		height: height,
	}, nil
}

func (g *Graphics) Reset() error {
	return nil
}

func (g *Graphics) Draw(indexLen int, indexOffset int, mode driver.CompositeMode, colorM *affine.ColorM, filter driver.Filter, address driver.Address) error {
	g.game.Draw(indexLen, indexOffset, mode, colorM, filter, address)
	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	panic("monogame: SetVsyncEnabled is not implemented yet")
}

func (g *Graphics) VDirection() driver.VDirection {
	return driver.VUpward
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

type screen struct {
	game *monogame.Game
}

func (s *screen) SetAsDestination(viewportWidth, viewportHeight int) {
	s.game.ResetDestination(viewportWidth, viewportHeight)
}

func (s *screen) SetAsSource() {
	panic("monogame: SetAsSource on screen is forbidden")
}

func (s *screen) ReplacePixels(args []*driver.ReplacePixelsArgs) {
	panic("monogame: ReplacePixels on screen is forbidden")
}

func (s *screen) Dispose() {
	// Do nothing?
}

func (s *screen) IsScreen() bool {
	return true
}
