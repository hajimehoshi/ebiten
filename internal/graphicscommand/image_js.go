//go:build js
// +build js

package graphicscommand

import (
	"bytes"
	"image"
	"path/filepath"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
	// Screen image cannot be dumped.
	if i.screen {
		return nil
	}

	path = i.DumpName(path)
	path = filepath.Base(path)

	global := js.Global()

	buf := &bytes.Buffer{}
	if err := i.DumpTo(graphicsDriver, blackbg, rect, buf); err != nil {
		return err
	}

	jsData := global.Get("Uint8Array").New(buf.Len())
	_ = js.CopyBytesToJS(jsData, buf.Bytes())

	a := global.Get("document").Call("createElement", "a")
	blob := global.Get("Blob").New(
		[]interface{}{jsData},
		map[string]interface{}{"type": "image/png"},
	)
	a.Set("href", global.Get("URL").Call("createObjectURL", blob))
	a.Set("download", path)
	a.Call("click")

	return nil
}

