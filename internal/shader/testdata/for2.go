package main

func Foo() vec2 {
	v := vec2(0)
	for i := 0; i < 100; i++ {
		v.x += i
		if v.x >= 100 {
			break
		}
	}
	v2 := vec2(0)
	for i := 10.0; i >= 0; i -= 2 {
		if v2.x < 100 {
			continue
		}
		v2.x += i
	}
	return v
}
