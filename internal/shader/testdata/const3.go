package main

const a = 1
const b float = 1
const c int = 1
const d, e = 1, 1.0
const f = false

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
