// Copyright 2026 The Ebitengine Authors
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
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector/internal/vectorpaintshader"
)

// Protected by theFillPathM.
var imagePaintShaders [vectorpaintshader.CoverageCount][2][2]*ebiten.Shader

func ensureImagePaintShader(rule FillRule, aa, clipped bool, filter ebiten.Filter) *ebiten.Shader {
	var antialiasIndex, filterIndex int
	var coverage vectorpaintshader.Coverage
	switch {
	case clipped:
		coverage = vectorpaintshader.CoverageClipped
	case rule == FillRuleNonZero:
		coverage = vectorpaintshader.CoverageNonZero
	case rule == FillRuleEvenOdd:
		coverage = vectorpaintshader.CoverageEvenOdd
	default:
		panic(fmt.Sprintf("vector: invalid fill rule: %d", rule))
	}
	if aa {
		antialiasIndex = 1
	}
	if filter == ebiten.FilterLinear {
		filterIndex = 1
	}
	shader := &imagePaintShaders[coverage][antialiasIndex][filterIndex]
	if *shader != nil {
		return *shader
	}
	source := vectorpaintshader.ShaderSource(coverage, aa, filter == ebiten.FilterLinear)
	s, err := ebiten.NewShader(source)
	if err != nil {
		panic(fmt.Sprintf("vector: failed to create image paint shader: %v", err))
	}
	*shader = s
	return s
}
