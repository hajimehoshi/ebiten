package main

func Foo() vec2 {
	var a, b vec2
	a, b = b, a
	var c, d, e vec2
	c, d, e = d, e, c
	return a
}
