package main

func Vertex(dstPos vec2, src0Pos vec2, color vec4) (vec4, vec2, vec4) {
	dstPos -= vec2(1)
	src0Pos.x = 0
	src0Pos[1]++
	dstPos, src0Pos = src0Pos, dstPos
	return vec4(dstPos, 0, 1), src0Pos, color
}

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	dstPos = vec4(0)
	src0Pos.x += 1
	color[3] *= 2
	color--
	return vec4(dstPos.x, src0Pos.y, color.z, color.w)
}
