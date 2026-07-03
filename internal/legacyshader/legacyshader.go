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

// Package legacyshader handles the texel-unit Kage shaders kept for backward compatibility. It converts a
// texel-unit shader source to an equivalent pixel-unit source before handing it to the pixel-only core, so
// the notion of texels never reaches the core.
package legacyshader

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// Unit is the coordinate unit a Kage shader is authored in, selected by the //kage:unit directive.
type Unit int

const (
	Texels Unit = iota
	Pixels
)

// ParseCompilerDirectives returns the unit selected by the //kage:unit directive, defaulting to texels.
func ParseCompilerDirectives(src []byte) (Unit, error) {
	value, err := graphics.ParseKageUnitDirective(src)
	if err != nil {
		return 0, err
	}
	switch value {
	case "", "texels":
		return Texels, nil
	case "pixels":
		return Pixels, nil
	default:
		return 0, fmt.Errorf("legacyshader: invalid value for //kage:unit: %s", value)
	}
}

// CompileShader compiles a Kage fragment shader source, in either unit, into an intermediate
// representation, and reports the unit the source is authored in.
func CompileShader(fragmentSrc []byte) (*shaderir.Program, Unit, error) {
	unit, err := ParseCompilerDirectives(fragmentSrc)
	if err != nil {
		return nil, 0, err
	}
	src := fragmentSrc
	if unit == Texels {
		src, err = convertToPixels(fragmentSrc)
		if err != nil {
			return nil, 0, err
		}
	}
	ir, err := graphics.CompileShader(src)
	if err != nil {
		return nil, 0, err
	}
	return ir, unit, nil
}

// CalcSourceID returns the source ID of a Kage fragment shader source, in either unit.
func CalcSourceID(fragmentSrc []byte) (shaderir.SourceID, error) {
	unit, err := ParseCompilerDirectives(fragmentSrc)
	if err != nil {
		return shaderir.SourceID{}, err
	}
	src := fragmentSrc
	if unit == Texels {
		src, err = convertToPixels(fragmentSrc)
		if err != nil {
			return shaderir.SourceID{}, err
		}
	}
	return graphics.CalcSourceID(src), nil
}
