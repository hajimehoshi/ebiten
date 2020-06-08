package main

func Foo(x vec2) {
	Bar(Bar(x.x, x.y))
}

func Bar(x, y float) (float, float) {
	return x, y
}
