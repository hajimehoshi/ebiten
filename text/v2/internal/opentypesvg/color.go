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

package opentypesvg

import (
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgcss"
)

// rgba contains straight sRGB components in [0, 1].
type rgba struct{ R, G, B, A float64 }

// colorContext supplies host colors before any SVG opacity is applied.
type colorContext struct {
	Foreground rgba
	Palette    []rgba
}

func (c rgba) isValid() bool {
	return areValidNumbers(c.R, c.G, c.B, c.A) && c.R >= 0 && c.R <= 1 && c.G >= 0 && c.G <= 1 && c.B >= 0 && c.B <= 1 && c.A >= 0 && c.A <= 1
}

// parseColor accepts a concrete color or currentColor with an explicit context.
func parseColor(text string, current rgba, context colorContext) (rgba, error) {
	if len(text) > maxStyleValue {
		return rgba{}, errLimit
	}
	text = strings.TrimSpace(text)
	// A fallback can itself contain var(), but each iteration consumes input.
	for depth := 0; svgcss.HasFunction(text, "var"); depth++ {
		if depth >= maxColorDepth {
			return rgba{}, errLimit
		}
		variable, err := svgcss.ParseVariable(text)
		if err != nil {
			return rgba{}, cssError(err)
		}
		if variable.Trailing != "" {
			return rgba{}, errStyle
		}
		if suffix, ok := strings.CutPrefix(variable.Name, "--color"); ok && suffix != "" && (len(suffix) == 1 || suffix[0] != '0') {
			index, err := strconv.ParseUint(suffix, 10, 32)
			if err == nil && strconv.FormatUint(index, 10) == suffix && index < uint64(len(context.Palette)) {
				return context.Palette[index], nil
			}
		}
		if !variable.HasFallback {
			return rgba{}, errStyle
		}
		text = strings.TrimSpace(variable.Fallback)
	}
	text = strings.ToLower(text)
	if text == "currentcolor" {
		return current, nil
	}
	if text == "transparent" {
		return rgba{}, nil
	}
	if c, ok := colornames.Map[text]; ok {
		return rgba{R: float64(c.R) / 255, G: float64(c.G) / 255, B: float64(c.B) / 255, A: 1}, nil
	}
	if strings.HasPrefix(text, "#") {
		hex := text[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) != 6 {
			return rgba{}, errStyle
		}
		n, err := strconv.ParseUint(hex, 16, 24)
		if err != nil {
			return rgba{}, errStyle
		}
		return rgba{R: float64(n>>16) / 255, G: float64((n>>8)&255) / 255, B: float64(n&255) / 255, A: 1}, nil
	}
	name, args, ok := strings.Cut(text, "(")
	if !ok || !strings.HasSuffix(args, ")") {
		return rgba{}, errStyle
	}
	args = args[:len(args)-1]
	fields := strings.Split(args, ",")
	count := 3
	if name == "rgba" || name == "hsla" {
		count = 4
	}
	if len(fields) != count {
		return rgba{}, errStyle
	}
	var v [4]float64
	v[3] = 1
	for i, s := range fields {
		s = strings.TrimSpace(s)
		percent := strings.HasSuffix(s, "%")
		n, err := parseNumber(strings.TrimSuffix(s, "%"))
		if err != nil {
			return rgba{}, errStyle
		}
		if percent {
			n /= 100
		}
		switch name {
		case "rgb", "rgba":
			if i < 3 && !percent {
				n /= 255
			}
		case "hsl", "hsla":
			if (i == 0 && percent) || ((i == 1 || i == 2) && !percent) {
				return rgba{}, errStyle
			}
		default:
			return rgba{}, errStyle
		}
		v[i] = n
	}
	if name == "hsl" || name == "hsla" {
		h := math.Mod(v[0], 360) / 60
		if h < 0 {
			h += 6
		}
		s, l := min(1, max(0, v[1])), min(1, max(0, v[2]))
		c := (1 - math.Abs(2*l-1)) * s
		x := c * (1 - math.Abs(math.Mod(h, 2)-1))
		m := l - c/2
		var r, g, b float64
		switch {
		case h < 1:
			r, g = c, x
		case h < 2:
			r, g = x, c
		case h < 3:
			g, b = c, x
		case h < 4:
			g, b = x, c
		case h < 5:
			r, b = x, c
		default:
			r, b = c, x
		}
		return rgba{R: min(1, max(0, r+m)), G: min(1, max(0, g+m)), B: min(1, max(0, b+m)), A: min(1, max(0, v[3]))}, nil
	}
	return rgba{R: min(1, max(0, v[0])), G: min(1, max(0, v[1])), B: min(1, max(0, v[2])), A: min(1, max(0, v[3]))}, nil
}
