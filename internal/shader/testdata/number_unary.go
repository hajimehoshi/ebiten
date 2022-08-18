package main

func Foo1() vec4 {
	x0 := +(+5)
	x1 := +(-5)
	x2 := -(+5)
	x3 := -(-5)
	x4 := +(+5.0)
	x5 := +(-5.0)
	x6 := -(+5.0)
	x7 := -(-5.0)
	return vec4(x0, x1, x2, x3) + vec4(x4, x5, x6, x7)
}

func Foo2() vec4 {
	x0 := +(+5)
	x1 := +(-5)
	x2 := -(+5)
	x3 := -(-5)
	x4 := +(+5.0)
	x5 := +(-5.0)
	x6 := -(+5.0)
	x7 := -(-5.0)
	return vec4(x0, x1, x2, x3) + vec4(x4, x5, x6, x7)
}
