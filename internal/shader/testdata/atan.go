package main

func Foo(x, y float) bool {
	a := atan(y / x)
	b := atan2(y, x)
	return a == b
}
