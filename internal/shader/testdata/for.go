package main

func Foo() vec2 {
	v := vec2(0)
	for i := 0; i < 100; i++ {
		v.x += float(i)
	}
	v2 := vec2(0)
	for i := 10.0; i >= 0; i -= 2 {
		v2.x += float(i)
	}
	_ = v2
	return v
}
