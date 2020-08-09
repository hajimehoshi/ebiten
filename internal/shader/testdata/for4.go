package main

func Ident(x int) int {
	return x
}

func Foo() int {
	sum := 0
	for i := 0; i < 10; i++ {
		x := Ident(i)
		sum += x
	}
	y := 0
	sum += y
	return sum
}
