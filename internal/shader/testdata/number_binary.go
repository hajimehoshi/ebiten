package main

func Foo1() vec4 {
	x0 := 5 / 2
	x1 := 5.0 / 2
	x2 := 5 / 2.0
	x3 := 5.0 / 2.0
	return vec4(x0, x1, x2, x3)
}

func Foo2() vec4 {
	var x0 = 5 / 2
	var x1 = 5.0 / 2
	var x2 = 5 / 2.0
	var x3 = 5.0 / 2.0
	return vec4(x0, x1, x2, x3)
}
