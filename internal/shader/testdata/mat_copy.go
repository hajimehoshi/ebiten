package main

func Foo(m mat2) mat2 {
	return mat2(m)
}

func Bar(m mat3) mat3 {
	return mat3(m)
}

func Baz(m mat4) mat4 {
	return mat4(m)
}

func Qux(x float) (mat2, mat3, mat4) {
	return mat2(x), mat3(x), mat4(x)
}
