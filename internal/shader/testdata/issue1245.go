package main

func Vertex(position vec2) vec4 {
	var v vec4
	for i := 0.0; i < 4.0; i++ {
		v.x += i * 0.01
	}
	return v
}

func Fragment(position vec4) vec4 {
	var v vec4
	for i := 0.0; i < 4.0; i++ {
		v.x += i * 0.01
	}
	return v
}
