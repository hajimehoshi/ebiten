package main

var Array [4]vec2

func Foo() [2]vec2 {
	var x [2]vec2
	return x
}

func Bar() [2]vec2 {
	x := [2]vec2{vec2(1)}
	x[1].y = vec2(2)
	return x
}
