package main

func Foo() vec2 {
	v := vec2(0)
	for i := 0; i < 100; i++ {
		v2 := vec2(0)
		v = v2
	}
	v3 := vec2(0)
	for i := 0; i < 100; i++ {
		v4 := vec2(0)
		v3 = v4
	}
	_ = v3
	return v
}
