package main

func Foo(foo vec2) vec4 {
	var r1, r2 float
	r1, r2 = Bar()
	return vec4(foo, r1, r2)
}

func Bar() (float, float) {
	return 0, 0
}
