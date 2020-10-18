// +build shader
package main

var Time float

func Distort(p vec2) vec2 {
	//theta := atan(p.y, p.x)
	//radius := pow(length(p), 1.1)

	//p.x = radius * cos(theta)
	//p.y = radius * sin(theta)

	//return 0.5 * (p + vec2(1.0, 1.0))
	warpX := 0.031
	warpY := 0.041
	p = p*2.0 - 1.0
	p *= vec2(1.0+(p.y*p.y)*warpX, 1.0+(p.x*p.x)*warpY)
	p.y += 0.5
	p.x += 0.51
	return p*0.5 + 0.25
}

func get_color_bleeding(current_color vec4, color_left vec4) (vec4, vec4) {
	color_bleeding := 1.2
	current_color = current_color * vec4(color_bleeding, 0.5, 1.0-color_bleeding, 1)
	color_left = color_left * vec4(1.0-color_bleeding, 0.5, color_bleeding, 1)
	return current_color, color_left
}

func get_color_scanline(uv vec2, c vec4, time float) vec4 {
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
	xy = Distort(xy)

	/*d := length(xy)
	if d < 1.5 {
		xy = Distort(xy)
	} else {
		xy = texCoord
	}*/

	bleeding_range_x := 2.0
	bleeding_range_y := 2.0
	screen_height := 480.0
	screen_width := 640.0

	pixel_size_x := 1.0 / screen_width * bleeding_range_x
	pixel_size_y := 1.0 / screen_height * bleeding_range_y
	color_left := imageSrc0At(xy - vec2(pixel_size_x, pixel_size_y))
	current_color := imageSrc0At(xy)
	color_left, current_color = get_color_bleeding(current_color, color_left)
	c := current_color + color_left
	return get_color_scanline(xy, c, Time)
}
