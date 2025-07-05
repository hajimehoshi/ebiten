// Copyright 2025 The Ebitengine Authors
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

package vector

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// LineCap represents the way in which how the ends of the stroke are rendered.
type LineCap int

const (
	LineCapButt LineCap = iota
	LineCapRound
	LineCapSquare
)

// LineJoin represents the way in which how two segments are joined.
type LineJoin int

const (
	LineJoinMiter LineJoin = iota
	LineJoinBevel
	LineJoinRound
)

// StrokeOptions is options to render a stroke.
type StrokeOptions struct {
	// Width is the stroke width in pixels.
	//
	// The default (zero) value is 0.
	Width float32

	// LineCap is the way in which how the ends of the stroke are rendered.
	// Line caps are not rendered when the sub-path is marked as closed.
	//
	// The default (zero) value is LineCapButt.
	LineCap LineCap

	// LineJoin is the way in which how two segments are joined.
	//
	// The default (zero) value is LineJoiMiter.
	LineJoin LineJoin

	// MiterLimit is the miter limit for LineJoinMiter.
	// For details, see https://developer.mozilla.org/en-US/docs/Web/SVG/Attribute/stroke-miterlimit.
	//
	// The default (zero) value is 0.
	MiterLimit float32
}

// AddPathStrokeOptions is options for [Path.AddPathStroke].
type AddPathStrokeOptions struct {
	// StrokeOptions is options for the stroke.
	StrokeOptions

	// GeoM is a geometry matrix to apply to the path.
	//
	// The default (zero) value is an identity matrix.
	GeoM ebiten.GeoM
}
