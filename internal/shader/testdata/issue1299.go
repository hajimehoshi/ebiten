package main

func Foo() int {
	x := 1
	x *= 2
	x = x + 2
	x = 2 - x
	return x
}
