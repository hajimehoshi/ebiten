package main

func Foo() vec2 {
	x := true
	{
		x := 0
		return vec2(x)
	}
	return vec2(1)
}
