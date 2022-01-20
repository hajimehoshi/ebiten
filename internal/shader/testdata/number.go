package main

func Foo() vec2 {
	x := Float() * 1
	y := 1 * Float()
	return vec2(x, y)
}

func Foo2() vec2 {
	var x = Float() * 1
	var y = 1 * Float()
	return vec2(x, y)
}

func Float() float {
	return 1.0
}

func TakeFloat(x float) {
}

func TakeInt(x int) {
}

func Foo3() {
	TakeFloat(1.0)
	TakeInt(1.0)
}
