package main

func Foo(foo vec2) int {
	var a [2]int
	return len(a)
}

func Bar(foo vec2) int {
	var a [3]int
	return cap(a)
}
