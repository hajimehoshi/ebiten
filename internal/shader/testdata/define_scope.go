package main

func Foo(x float) float {
	y := 0.0
	{
		x, z := 1.0, x
		y = x + z
	}
	return y
}
