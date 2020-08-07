package main

func Vertex(position vec2, texCoord vec2, color vec4) (position vec4, texCoord vec2, color vec4) {
	projectionMatrix := mat4(
		2/ScreenSize.x, 0, 0, 0,
		0, 2/ScreenSize.y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	)
	return projectionMatrix * vec4(position, 0, 1), texCoord, color
}

var ScreenSize vec2
