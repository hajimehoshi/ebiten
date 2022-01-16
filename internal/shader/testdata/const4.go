package main

func Foo(x int) int {
	const a = 1
	return a + x
}

func Bar(x float) float {
	const a = 1
	return a + x
}

func Baz() int {
	const a = 1
	return Foo(a)
}

func Qux() float {
	const a = 1
	return Bar(a)
}
