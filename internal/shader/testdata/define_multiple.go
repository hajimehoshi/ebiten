package main

func Foo(foo vec2) vec4 {
	r1, r2 := Bar()
	_ = r2
	return vec4(foo, r1, r1)
}

func Bar() (float, float) {
	return 0, 0
}
