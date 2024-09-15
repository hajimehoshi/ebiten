// Copyright 2019 The Ebiten Authors
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// CursorModeType represents a render and coordinate mode of a mouse cursor.
type CursorModeType int

// CursorModeTypes
const (
	CursorModeVisible  CursorModeType = CursorModeType(ui.CursorModeVisible)
	CursorModeHidden   CursorModeType = CursorModeType(ui.CursorModeHidden)
	CursorModeCaptured CursorModeType = CursorModeType(ui.CursorModeCaptured)
)

// CursorShapeType represents a shape of a mouse cursor.
type CursorShapeType int

// CursorShapeTypes
const (
	CursorShapeDefault    CursorShapeType = CursorShapeType(ui.CursorShapeDefault)
	CursorShapeText       CursorShapeType = CursorShapeType(ui.CursorShapeText)
	CursorShapeCrosshair  CursorShapeType = CursorShapeType(ui.CursorShapeCrosshair)
	CursorShapePointer    CursorShapeType = CursorShapeType(ui.CursorShapePointer)
	CursorShapeEWResize   CursorShapeType = CursorShapeType(ui.CursorShapeEWResize)
	CursorShapeNSResize   CursorShapeType = CursorShapeType(ui.CursorShapeNSResize)
	CursorShapeNESWResize CursorShapeType = CursorShapeType(ui.CursorShapeNESWResize)
	CursorShapeNWSEResize CursorShapeType = CursorShapeType(ui.CursorShapeNWSEResize)
	CursorShapeMove       CursorShapeType = CursorShapeType(ui.CursorShapeMove)
	CursorShapeNotAllowed CursorShapeType = CursorShapeType(ui.CursorShapeNotAllowed)
)

// CursorShape returns the current cursor shape.
//
// CursorShape returns CursorShapeDefault on mobiles.
//
// CursorShape is concurrent-safe.
func CursorShape() CursorShapeType {
	return CursorShapeType(ui.Get().CursorShape())
}

// SetCursorShape sets the cursor shape.
//
// If the platform doesn't implement the given shape, the default cursor shape is used.
//
// SetCursorShape is concurrent-safe.
func SetCursorShape(shape CursorShapeType) {
	ui.Get().SetCursorShape(ui.CursorShape(shape))
}
