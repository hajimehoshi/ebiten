// Copyright 2020 The Ebiten Authors
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

// +build ignore

package main

var Time float

func distort(p vec2) vec2 {
	warpX := 0.031
	warpY := 0.041
	p = p*2.0 - 1.0
	p *= vec2(1.0+(p.y*p.y)*warpX, 1.0+(p.x*p.x)*warpY)
	p.y += 0.5
	p.x += 0.51
	return p*0.5 + 0.25
}

func colorBleeding(current_color vec4, color_left vec4) (vec4, vec4) {
	color_bleeding := 1.2
	current_color = current_color * vec4(color_bleeding, 0.5, 1.0-color_bleeding, 1)
	color_left = color_left * vec4(1.0-color_bleeding, 0.5, color_bleeding, 1)
	return current_color, color_left
}

func colorScanline(uv vec2, c vec4, time float) vec4 {
	screen_height := 480.0
	scan_size := 2.0
	lines_velocity := 30.0
	scanline_alpha := 0.9
	lines_distance := 4.0
	line_row := floor((uv.y * screen_height / scan_size) + mod(time*lines_velocity, lines_distance))
	n := 1.0 - ceil((mod(line_row, lines_distance) / lines_distance))
	c = c - n*c*(1.0-scanline_alpha)
	c.a = 1.0
	return c
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	xy := texCoord
	xy = distort(xy)

	bleeding_range_x := 2.0
	bleeding_range_y := 2.0
	screen_height := 480.0
	screen_width := 640.0

	pixel_size_x := 1.0 / screen_width * bleeding_range_x
	pixel_size_y := 1.0 / screen_height * bleeding_range_y
	color_left := imageSrc0At(xy - vec2(pixel_size_x, pixel_size_y))
	current_color := imageSrc0At(xy)
	color_left, current_color = colorBleeding(current_color, color_left)
	c := current_color + color_left
	return colorScanline(xy, c, Time)
}
