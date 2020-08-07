package main

func Foo(foo vec2) vec4 {
	var bar1 vec4 = vec4(foo, 0, 1)
	bar1.x = bar1.x
	var bar2 vec4 = bar1
	return bar2
}
