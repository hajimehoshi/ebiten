// Copyright 2015 Hajime Hoshi
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
	"image/color"
)

// A Lines represents the set of lines.
type Lines interface {
	Len() int
	Points(i int) (x0, y0, x1, y1 int)
	Color(i int) color.Color
}

type line struct {
	x0, y0 int
	x1, y1 int
	color  color.Color
}

func (l *line) Len() int {
	return 1
}

func (l *line) Points(i int) (x0, y0, x1, y1 int) {
	return l.x0, l.y0, l.x1, l.y1
}

func (l *line) Color(i int) color.Color {
	return l.color
}

type rectsAsLines struct {
	Rects
}

func (r *rectsAsLines) Len() int {
	return r.Rects.Len() * 4
}

func (r *rectsAsLines) Points(i int) (x0, y0, x1, y1 int) {
	x, y, w, h := r.Rects.Rect(i / 4)
	switch i % 4 {
	case 0:
		return x, y, x + w, y
	case 1:
		return x, y + 1, x, y + h - 1
	case 2:
		return x, y + h - 1, x + w, y + h - 1
	case 3:
		return x + w - 1, y + 1, x + w - 1, y + h - 1
	}
	panic("not reach")
}

func (r *rectsAsLines) Color(i int) color.Color {
	return r.Rects.Color(i / 4)
}

// A Rects represents the set of rectangles.
type Rects interface {
	Len() int
	Rect(i int) (x, y, width, height int)
	Color(i int) color.Color
}

type rect struct {
	x, y          int
	width, height int
	color         color.Color
}

func (r *rect) Len() int {
	return 1
}

func (r *rect) Rect(i int) (x, y, width, height int) {
	return r.x, r.y, r.width, r.height
}

func (r *rect) Color(i int) color.Color {
	return r.color
}
