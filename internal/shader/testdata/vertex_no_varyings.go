package main

func Vertex(dstPos vec2, src0Pos vec2, color vec4) vec4 {
	return vec4(dstPos, 0, 1)
}

func Fragment() vec4 {
	return vec4(1)
}
