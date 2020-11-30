package main

func Foo() (a float, b [4]float, c vec4) {
	return
}

func Foo2() (a float, b [4]float, c vec4) {
	return Foo()
}
