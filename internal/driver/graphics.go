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

package driver

import (
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

type Graphics interface {
	SetThread(thread *thread.Thread)
	Begin()
	End()
	SetTransparent(transparent bool)
	SetVertices(vertices []float32, indices []uint16)
	NewImage(width, height int) (Image, error)
	NewScreenFramebufferImage(width, height int) (Image, error)
	Reset() error
	Draw(indexLen int, indexOffset int, mode CompositeMode, colorM *affine.ColorM, filter Filter, address Address) error
	SetVsyncEnabled(enabled bool)
	VDirection() VDirection
	NeedsRestoring() bool
	IsGL() bool
	HasHighPrecisionFloat() bool
	MaxImageSize() int
}

type Image interface {
	Dispose()
	IsInvalidated() bool
	Pixels() ([]byte, error)
	SetAsDestination()
	SetAsSource()
	ReplacePixels(args []*ReplacePixelsArgs)
}

type ReplacePixelsArgs struct {
	Pixels []byte
	X      int
	Y      int
	Width  int
	Height int
}

type VDirection int

const (
	VUpward VDirection = iota
	VDownward
)
