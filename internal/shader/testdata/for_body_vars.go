package main

func Foo(a float) float {
	sum := 0.0
	for i := 0; i < 4; i++ {
		var x float
		if y := float(i) * a; y > 1 {
			x = y * 2
			sum += x
		}
	}
	for i := 0; i < 4; i++ {
		var x float
		{
			x = float(i) * a
			sum += x
		}
	}
	for i := range 4 {
		var x float
		if y := float(i) * a; y > 1 {
			x = y * 2
			sum += x
		}
	}
	var arr [2]float
	for _, v := range arr {
		var x float
		{
			x = v * a
			sum += x
		}
	}
	return sum
}
