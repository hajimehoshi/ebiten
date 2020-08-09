package main

var (
	U0 float
	U1 float
	U2 float
)

func Ident(x int) int {
	return x
}

func Vertex(pos vec2) vec4 {
	sum := 0
	for i := 0; i < 10; i++ {
		x := Ident(i)
		sum += x
		for j := 0; j < 10; j++ {
			x := Ident(j)
			sum += x
		}
	}
	y := 0
	sum += y
	return vec4(sum)
}

func Fragment(pos vec4) vec4 {
	sum := 0
	for i := 0; i < 10; i++ {
		x := Ident(i)
		sum += x
		for j := 0; j < 10; j++ {
			x := Ident(j)
			sum += x
		}
	}
	y := 0
	sum += y
	return vec4(sum)
}
