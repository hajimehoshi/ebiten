package main

func Foo(foo vec2) vec4 {
	var r1 float
	r1 = 0.0
	r1 += 1.0
	return vec4(foo, r1, r1)
}
