package main

func Foo() vec2 {
	x0 := 1 * Bar()
	x1 := Bar() * 1
	_ = x1
	return x0
}

func Bar() vec2 {
	return vec2(0)
}
