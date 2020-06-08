package main

func Foo(foo vec2) vec4 {
	var bar1, bar2 vec2 = foo, foo
	return vec4(bar1, bar2)
}
