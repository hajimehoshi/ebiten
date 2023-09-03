package main

func Ret1() float {
	var a float
	{
		a = 0
	}
	return a
}

func Ret2() (float, vec2) {
	var a float
	var b vec2
	{
		a, b = 0, vec2(0)
	}
	return a, b
}

func Ret2Plus(foo bool) (float, vec2) {
	var a float
	var b vec2
	{
		a, b = 0, vec2(0)
	}
	return a, b
}
