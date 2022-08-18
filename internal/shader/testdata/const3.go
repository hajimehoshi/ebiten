package main

const (
	a          = 1
	b    float = 1
	c    int   = 1
	d, e       = 1, 1.0
	f          = false
)

func Foo() {
	l0 := 1
	la := a
	lb := b
	lc := c
	ld, le := d, e
	lf := f
	_ = l0
	_ = la
	_ = lb
	_ = lc
	_ = ld
	_ = le
	_ = lf
}
