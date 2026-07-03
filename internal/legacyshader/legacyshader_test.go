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

package legacyshader_test

import (
	"bufio"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/legacyshader"
)

func TestCompilerDirective(t *testing.T) {
	cases := []struct {
		src  string
		unit legacyshader.Unit
		err  bool
	}{
		{
			src: `package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: legacyshader.Texels,
			err:  false,
		},
		{
			src: `//kage:unit texels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: legacyshader.Texels,
			err:  false,
		},
		{
			src: `//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: legacyshader.Pixels,
			err:  false,
		},
		{
			src: `//kage:unit foo

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			err: true,
		},
		{
			src: `//kage:unit pixels
//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			err: true,
		},
		{
			src: `//kage:unit pixels
//kage:unit texels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			err: true,
		},
		{
			src: "\t    " + `//kage:unit pixels` + "    \t\r" + `
package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: legacyshader.Pixels,
			err:  false,
		},
		{
			// A directive must be parsed even after a line longer than bufio.MaxScanTokenSize.
			src: `// ` + strings.Repeat("a", bufio.MaxScanTokenSize) + `
//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return dstPos
}`,
			unit: legacyshader.Pixels,
			err:  false,
		},
	}
	for _, c := range cases {
		unit, err := legacyshader.ParseCompilerDirectives([]byte(c.src))
		if err == nil && c.err {
			t.Errorf("ParseCompilerDirectives(%q) must return an error but does not", c.src)
		} else if err != nil && !c.err {
			t.Errorf("ParseCompilerDirectives(%q) must not return an error but returned %v", c.src, err)
		}
		if err != nil || c.err {
			continue
		}
		if got, want := unit, c.unit; got != want {
			t.Errorf("ParseCompilerDirectives(%q): got: %d, want: %d", c.src, got, want)
		}
		if _, unit, err := legacyshader.CompileShader([]byte(c.src)); err != nil {
			t.Errorf("CompileShader(%q) must not return an error but returned %v", c.src, err)
		} else if got, want := unit, c.unit; got != want {
			t.Errorf("CompileShader(%q): unit: got: %d, want: %d", c.src, got, want)
		}
	}
}

func TestConvertToPixels(t *testing.T) {
	src := []byte(`//kage:unit texels

package main

func origins() (vec2, vec2) {
	return imageSrc0Origin(), imageSrc1Origin()
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	orig, size := imageSrcRegionOnTexture()
	dstOrig, dstSize := imageDstRegionOnTexture()
	a, b := origins()
	pos := imageSrc1Size() + imageDstOrigin() + imageDstSize() + orig + size + dstOrig + dstSize + a + b
	return imageSrc0At(srcPos) + (imageSrc1UnsafeAt)(pos) + vec4(imageSrcTextureSize().x, imageDstTextureSize().y, 0, 0)*0
}`)

	out, err := legacyshader.ConvertToPixels(src)
	if err != nil {
		t.Fatal(err)
	}
	outStr := string(out)

	for _, want := range []string{
		"//kage:unit pixels",
		"__legacyshader_imageSrc0At(srcPos)",
		"__legacyshader_imageSrc1UnsafeAt",
		"__legacyshader_imageSrcRegionOnTexture()",
		"__legacyshader_imageDstRegionOnTexture()",
		"__legacyshader_imageSrc0Origin(), __legacyshader_imageSrc1Origin()",
		"func Fragment(dstPos vec4, __legacyshader_srcPos vec2, color vec4) vec4 {",
		"srcPos := __legacyshader_srcPos / max(__imageSrcTextureSizes[0], vec2(1))",
		"_ = srcPos",
	} {
		if !strings.Contains(outStr, want) {
			t.Errorf("ConvertToPixels result must contain %q but does not:\n%s", want, outStr)
		}
	}
	for _, wantNot := range []string{
		"kage:unit texels",
		"__legacyshader_imageSrcTextureSize",
		"__legacyshader_imageDstTextureSize()",
	} {
		if strings.Contains(outStr, wantNot) {
			t.Errorf("ConvertToPixels result must not contain %q but does:\n%s", wantNot, outStr)
		}
	}

	// The converted source must be a valid pixel-unit shader.
	if _, err := graphics.CompileShader(out); err != nil {
		t.Errorf("CompileShader must not return an error but returned %v\nsource:\n%s", err, outStr)
	}
}

func TestConvertToPixelsFragmentVariants(t *testing.T) {
	cases := []string{
		`package main

func Fragment(dstPos vec4) vec4 {
	return dstPos
}`,
		`package main

func Fragment() vec4 {
	return vec4(0)
}`,
		`package main

func Fragment(dstPos vec4, srcPos vec2) vec4 {
	return imageSrc0At(srcPos)
}`,
		`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	return imageSrc0At(srcPos) + color + custom
}`,
		`package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return color
}`,
	}
	for _, src := range cases {
		if _, _, err := legacyshader.CompileShader([]byte(src)); err != nil {
			t.Errorf("CompileShader(%q) must not return an error but returned %v", src, err)
		}
	}
}
