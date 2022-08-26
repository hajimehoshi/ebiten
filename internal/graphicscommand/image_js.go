//go:build js
// +build js

package graphicscommand

import (
	"image"
	"path/filepath"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
	data, err := i.DumpBytes(graphicsDriver, blackbg, rect)
	if err != nil {
		return err
	}

	// screen image
	if data == nil {
		return nil
	}

	path = i.FormatPath(path)
	path = filepath.Base(path)

	global := js.Global()

	jsData := global.Get("Uint8Array").New(len(data))
	_ = js.CopyBytesToJS(jsData, data)

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

