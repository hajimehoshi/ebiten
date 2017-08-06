// +build js

package ui

import (
	"unicode"

	"github.com/gopherjs/gopherjs/js"
)

func keypress(e *js.Object) {
	if runebuffer != nil {
		if r := rune(e.Get("charCode").Int()); unicode.IsPrint(r) {
			runebuffer = append(runebuffer, r)
		}
	}
}

func Keyboard() []rune {
	if runebuffer == nil {
		runebuffer = make([]rune, 0, 1024)
	}
	rb := runebuffer
	runebuffer = runebuffer[:0]
	return rb
}
