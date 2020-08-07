package main

func Foo(foo vec2) vec4 {
	var bar1, bar2 vec2 = foo, foo
	return vec4(bar1, bar2)
}

func Foo2(foo vec2) vec4 {
	var bar1, bar2 = Bar()
	return vec4(bar1, bar2)
}

func Bar() (vec2, vec2) {
	return vec2(0), vec2(0)
}
