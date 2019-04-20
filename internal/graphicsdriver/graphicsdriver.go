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

package graphicsdriver

import (
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

type GraphicsDriver interface {
	Begin()
	End()
	SetWindow(window uintptr)
	SetVertices(vertices []float32, indices []uint16)
	Flush()
	NewImage(width, height int) (Image, error)
	NewScreenFramebufferImage(width, height int) (Image, error)
	Reset() error
	Draw(indexLen int, indexOffset int, mode graphics.CompositeMode, colorM *affine.ColorM, filter graphics.Filter, address graphics.Address) error
	SetVsyncEnabled(enabled bool)
	VDirection() VDirection
	IsGL() bool
}

type Image interface {
	Dispose()
	IsInvalidated() bool
	Pixels() ([]byte, error)
	SetAsDestination()
	SetAsSource()
	ReplacePixels(pixels []byte, x, y, width, height int)
}

type VDirection int

const (
	VUpward VDirection = iota
	VDownward
)
