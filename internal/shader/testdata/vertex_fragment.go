package main

func Vertex(dstPos vec2, srcPos vec2, color vec4) (dstPos vec4, srcPos vec2, color vec4) {
	projectionMatrix := mat4(
		2/ScreenSize.x, 0, 0, 0,
		0, 2/ScreenSize.y, 0, 0,
		0, 0, 1, 0,
		-1, -1, 0, 1,
	)
	return projectionMatrix * vec4(dstPos, 0, 1), srcPos, color
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(dstPos.x, srcPos.y, color.z, 1)
}

var ScreenSize vec2
