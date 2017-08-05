// +build js

package ui

import "github.com/gopherjs/gopherjs/js"

var doc = js.Global.Get("document")

func push(char rune) {
	go func() {
		rblock.Lock()
		runebuffer = append(runebuffer, char)
		rblock.Unlock()
	}()
}

func init() {
	if doc == nil {
		return
	}
	kp := doc.Get("onkeypress")
	switch kp.Length() {
	case 0:
		doc.Set("onkeypress", push)
	default:
		kp.Call("push", push)
	}
}
