package main

func Vertex(dstPos vec2, src0Pos vec2, color vec4) (dstPos vec4, src0Pos vec2, color vec4) {
	projectionMatrix := mat4(
		2/ScreenSize.x, 0, 0, 0,
		0, 2/ScreenSize.y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	)
	return projectionMatrix * vec4(dstPos, 0, 1), src0Pos, color
}

var ScreenSize vec2
