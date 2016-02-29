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
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

// Filter represents the type of filter to be used when an image is maginified or minified.
type Filter int

const (
	FilterNearest Filter = iota // nearest (crisp-edged) filter
	FilterLinear                // linear filter
)

func glFilter(c *opengl.Context, filter Filter) opengl.Filter {
	switch filter {
	case FilterNearest:
		return c.Nearest
	case FilterLinear:
		return c.Linear
	}
	panic("not reach")
}

// CompositeMode represents Porter-Duff composition mode.
type CompositeMode int

// Note: This name convention follow CSS compositing: https://drafts.fxtf.org/compositing-2/

const (
	CompositeModeSourceOver      CompositeMode = CompositeMode(opengl.CompositeModeSourceOver) // regular alpha blending
	CompositeModeClear                         = CompositeMode(opengl.CompositeModeClear)
	CompositeModeCopy                          = CompositeMode(opengl.CompositeModeCopy)
	CompositeModeDestination                   = CompositeMode(opengl.CompositeModeDestination)
	CompositeModeSourceIn                      = CompositeMode(opengl.CompositeModeSourceIn)
	CompositeModeDestinationIn                 = CompositeMode(opengl.CompositeModeDestinationIn)
	CompositeModeSourceOut                     = CompositeMode(opengl.CompositeModeSourceOut)
	CompositeModeDestinationOut                = CompositeMode(opengl.CompositeModeDestinationOut)
	CompositeModeSourceAtop                    = CompositeMode(opengl.CompositeModeSourceAtop)
	CompositeModeDestinationAtop               = CompositeMode(opengl.CompositeModeDestinationAtop)
	CompositeModeXor                           = CompositeMode(opengl.CompositeModeXor)
	CompositeModeLighter                       = CompositeMode(opengl.CompositeModeLighter) // sum of source and destination (a.k.a. 'plus' or 'additive')
)
