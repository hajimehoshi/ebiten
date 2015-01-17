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
	Points(i int) (x0, y0, x1, y1 int) // TODO: Change to float64?
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

// A Rects represents the set of rectangles.
type Rects interface {
	Len() int
	Rect(i int) (x, y, width, height int) // TODO: Change to float64?
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
