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

package legacyshader

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
)

// deprecatedFunctionsSuffix is the Kage source defining the deprecated builtin functions on top of
// their replacements in the core. A source image other than the 0th one is sampled by
// imageSrcNAtFromSrc0Pos and imageSrcNUnsafeAtFromSrc0Pos, and these are the same functions under the
// names they had before v2.10.
var deprecatedFunctionsSuffix = func() string {
	var b strings.Builder
	for i := 1; i < graphics.ShaderSrcImageCount; i++ {
		b.WriteString(fmt.Sprintf(`
// imageSrc%[1]dUnsafeAt returns the source image's color at the given position.
// The position is in the 0th source image's texture.
//
// Deprecated: as of v2.10. Use imageSrc%[1]dUnsafeAtFromSrc0Pos instead.
func imageSrc%[1]dUnsafeAt(pos vec2) vec4 {
	return imageSrc%[1]dUnsafeAtFromSrc0Pos(pos)
}

// imageSrc%[1]dAt returns the source image's color at the given position.
// The position is in the 0th source image's texture.
//
// Deprecated: as of v2.10. Use imageSrc%[1]dAtFromSrc0Pos instead.
func imageSrc%[1]dAt(pos vec2) vec4 {
	return imageSrc%[1]dAtFromSrc0Pos(pos)
}
`, i))
	}
	return b.String()
}()
