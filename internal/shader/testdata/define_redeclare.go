package main

func Foo(x float) vec2 {
	a := x
	a, b := 1.0, a
	return vec2(a, b)
}

func Foo2() vec2 {
	a := 1.0
	b, a := Bar()
	return vec2(a, b)
}

func Foo3(x float) float {
	x, y := 2.0, x
	return x + y
}

func Foo4() (r float) {
	r, s := 1.0, 2.0
	r += s
	return
}

func Bar() (float, float) {
	return 0, 0
}
