package main

func Foo() [3]vec2 {
	var a [2]vec2
	{
		var b [2]vec2
		b = a
		_ = b
	}
	var c [3]vec2
	return c
}
