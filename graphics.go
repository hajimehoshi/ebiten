// Copyright 2014 Hajime Hoshi
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
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

// Filter represents the type of texture filter to be used when an image is maginified or minified.
type Filter int

const (
	// FilterDefault represents the default filter.
	FilterDefault Filter = Filter(graphics.FilterDefault)

	// FilterNearest represents nearest (crisp-edged) filter
	FilterNearest Filter = Filter(graphics.FilterNearest)

	// FilterLinear represents linear filter
	FilterLinear Filter = Filter(graphics.FilterLinear)

	// filterScreen represents a special filter for screen. Inner usage only.
	//
	// Some parameters like a color matrix or color vertex values can be ignored when filterScreen is used.
	filterScreen Filter = Filter(graphics.FilterScreen)
)

// CompositeMode represents Porter-Duff composition mode.
type CompositeMode int

// This name convention follows CSS compositing: https://drafts.fxtf.org/compositing-2/.
//
// In the comments,
// c_src, c_dst and c_out represent alpha-premultiplied RGB values of source, destination and output respectively. α_src and α_dst represent alpha values of source and destination respectively.
const (
	// Regular alpha blending
	// c_out = c_src + c_dst × (1 - α_src)
	CompositeModeSourceOver CompositeMode = CompositeMode(graphics.CompositeModeSourceOver)

	// c_out = 0
	CompositeModeClear CompositeMode = CompositeMode(graphics.CompositeModeClear)

	// c_out = c_src
	CompositeModeCopy CompositeMode = CompositeMode(graphics.CompositeModeCopy)

	// c_out = c_dst
	CompositeModeDestination CompositeMode = CompositeMode(graphics.CompositeModeDestination)

	// c_out = c_src × (1 - α_dst) + c_dst
	CompositeModeDestinationOver CompositeMode = CompositeMode(graphics.CompositeModeDestinationOver)

	// c_out = c_src × α_dst
	CompositeModeSourceIn CompositeMode = CompositeMode(graphics.CompositeModeSourceIn)

	// c_out = c_dst × α_src
	CompositeModeDestinationIn CompositeMode = CompositeMode(graphics.CompositeModeDestinationIn)

	// c_out = c_src × (1 - α_dst)
	CompositeModeSourceOut CompositeMode = CompositeMode(graphics.CompositeModeSourceOut)

	// c_out = c_dst × (1 - α_src)
	CompositeModeDestinationOut CompositeMode = CompositeMode(graphics.CompositeModeDestinationOut)

	// c_out = c_src × α_dst + c_dst × (1 - α_src)
	CompositeModeSourceAtop CompositeMode = CompositeMode(graphics.CompositeModeSourceAtop)

	// c_out = c_src × (1 - α_dst) + c_dst × α_src
	CompositeModeDestinationAtop CompositeMode = CompositeMode(graphics.CompositeModeDestinationAtop)

	// c_out = c_src × (1 - α_dst) + c_dst × (1 - α_src)
	CompositeModeXor CompositeMode = CompositeMode(graphics.CompositeModeXor)

	// Sum of source and destination (a.k.a. 'plus' or 'additive')
	// c_out = c_src + c_dst
	CompositeModeLighter CompositeMode = CompositeMode(graphics.CompositeModeLighter)
)
