// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"errors"
)

type Context interface {
	UpdateFrame() error
	ForceUpdateFrame() error
	Layout(outsideWidth, outsideHeight float64)

	// AdjustPosition can be called from a different goroutine from Update's or Layout's.
	AdjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64)
}

type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
)

type TouchID int

// RegularTermination represents a regular termination.
// Run can return this error, and if this error is received,
// the game loop should be terminated as soon as possible.
var RegularTermination = errors.New("regular termination")

type FPSMode int

const (
	FPSModeVsyncOn FPSMode = iota
	FPSModeVsyncOffMaximum
	FPSModeVsyncOffMinimum
)

type CursorMode int

const (
	CursorModeVisible CursorMode = iota
	CursorModeHidden
	CursorModeCaptured
)

type CursorShape int

const (
	CursorShapeDefault CursorShape = iota
	CursorShapeText
	CursorShapeCrosshair
	CursorShapePointer
	CursorShapeEWResize
	CursorShapeNSResize
)
