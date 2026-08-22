package main

var U [4]vec2

func Foo() float {
	var a [3]float
	sum := 0.0
	for range a {
		sum += 1
	}
	for i := range a {
		sum += a[i]
	}
	for i, v := range a {
		sum += float(i) * v
	}
	for _, v := range U {
		sum += v.x
	}
	return sum
}
