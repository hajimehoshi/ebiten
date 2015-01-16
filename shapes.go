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
	"image/color"
)

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
