package main

func Vertex() (dstPos vec4, color vec4) {
	return vec4(0, 0, 0, 1), vec4(1)
}

func Fragment(dstPos vec4, color vec4) vec4 {
	return color
}
