package main

func Foo() int {
	sum := 0
	for i := range 10 {
		sum += i
	}
	const n = 4
	for i := range n {
		for j := range n {
			sum += i * j
		}
	}
	for range 3 {
		sum++
	}
	return sum
}
