package main

func Foo() (int, int) {
	return 1, 1
}

func Bar() {
	_, _ = Foo()
	a, _ := Foo()
	_, b := Foo()
	_, _ = a, b
}
