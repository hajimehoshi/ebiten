package main

func Vertex(dstPos vec2, src0Pos vec2, color vec4) (dstPos vec4, src0Pos vec2, color vec4) {
	return project(dstPos), src0Pos, half(color)
}

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return tint(color)
}

func project(pos vec2) vec4 {
	return vec4(pos, 0, 1)
}

func half(c vec4) vec4 {
	return c * 0.5
}

func tint(c vec4) vec4 {
	return half(c) * vec4(1, 0, 0, 1)
}
