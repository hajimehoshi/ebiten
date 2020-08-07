package main

func Foo(x vec2) vec2 {
	var xx, yx float = Bar(x.x, x.y)
	return vec2(xx, yx)
}

func Bar(x float) (float, float) {
	return x, x
}
