package main

func Foo() {
	Bar()
	Baz()
}

func Bar() {
	Baz()
	Quux()
}

func Baz() {
}

func Quux() {
	Baz()
}

func NeverCalled() {
	Foo()
	Bar()
	Baz()
	Quux()
}

func Vertex(pos vec2) vec4 {
	Foo()
	return vec4(0)
}

func Fragment(pos vec4) vec4 {
	Quux()
	return vec4(0)
}
