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
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

// Filter represents the type of texture filter to be used when an image is maginified or minified.
type Filter int

const (
	// FilterNearest represents nearest (crisp-edged) filter
	FilterNearest Filter = Filter(driver.FilterNearest)

	// FilterLinear represents linear filter
	FilterLinear Filter = Filter(driver.FilterLinear)

	// filterScreen represents a special filter for screen. Inner usage only.
	//
	// Some parameters like a color matrix or color vertex values can be ignored when filterScreen is used.
	filterScreen Filter = Filter(driver.FilterScreen)
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
	CompositeModeSourceOver CompositeMode = CompositeMode(driver.CompositeModeSourceOver)

	// c_out = 0
	CompositeModeClear CompositeMode = CompositeMode(driver.CompositeModeClear)

	// c_out = c_src
	CompositeModeCopy CompositeMode = CompositeMode(driver.CompositeModeCopy)

	// c_out = c_dst
	CompositeModeDestination CompositeMode = CompositeMode(driver.CompositeModeDestination)

	// c_out = c_src × (1 - α_dst) + c_dst
	CompositeModeDestinationOver CompositeMode = CompositeMode(driver.CompositeModeDestinationOver)

	// c_out = c_src × α_dst
	CompositeModeSourceIn CompositeMode = CompositeMode(driver.CompositeModeSourceIn)

	// c_out = c_dst × α_src
	CompositeModeDestinationIn CompositeMode = CompositeMode(driver.CompositeModeDestinationIn)

	// c_out = c_src × (1 - α_dst)
	CompositeModeSourceOut CompositeMode = CompositeMode(driver.CompositeModeSourceOut)

	// c_out = c_dst × (1 - α_src)
	CompositeModeDestinationOut CompositeMode = CompositeMode(driver.CompositeModeDestinationOut)

	// c_out = c_src × α_dst + c_dst × (1 - α_src)
	CompositeModeSourceAtop CompositeMode = CompositeMode(driver.CompositeModeSourceAtop)

	// c_out = c_src × (1 - α_dst) + c_dst × α_src
	CompositeModeDestinationAtop CompositeMode = CompositeMode(driver.CompositeModeDestinationAtop)

	// c_out = c_src × (1 - α_dst) + c_dst × (1 - α_src)
	CompositeModeXor CompositeMode = CompositeMode(driver.CompositeModeXor)

	// Sum of source and destination (a.k.a. 'plus' or 'additive')
	// c_out = c_src + c_dst
	CompositeModeLighter CompositeMode = CompositeMode(driver.CompositeModeLighter)

	// The product of source and destination (a.k.a 'multiply blend mode')
	// c_out = c_src * c_dst
	CompositeModeMultiply CompositeMode = CompositeMode(driver.CompositeModeMultiply)
)
