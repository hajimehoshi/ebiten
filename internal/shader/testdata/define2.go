package main

func Foo() vec2 {
	x := 1 * Bar()
	return x
}

func Bar() vec2 {
	return vec2(0)
}
