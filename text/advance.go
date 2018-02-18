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

package text

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var glyphAdvanceCache = map[font.Face]map[rune]fixed.Int26_6{}

func glyphAdvance(face font.Face, r rune) fixed.Int26_6 {
	m, ok := glyphAdvanceCache[face]
	if !ok {
		m = map[rune]fixed.Int26_6{}
		glyphAdvanceCache[face] = m
	}

	a, ok := m[r]
	if !ok {
		a, _ = face.GlyphAdvance(r)
		m[r] = a
	}

	return a
}
