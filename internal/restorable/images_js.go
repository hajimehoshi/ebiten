// Copyright 2017 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package restorable

import (
	"archive/zip"
	"bytes"
	"image"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// DumpImages dumps all the current images to the specified directory.
//
// This is for testing usage.
func DumpImages(graphicsDriver graphicsdriver.Graphics, dir string) error {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	for img := range theImages.images {
		f, err := zw.Create(img.image.DumpName("*.png"))
		if err != nil {
			return err
		}

		if err := img.image.DumpTo(f, graphicsDriver, false, image.Rect(0, 0, img.width, img.height)); err != nil {
			return err
		}
	}

	zw.Close()

	global := js.Global()

	jsData := global.Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(jsData, buf.Bytes())

	a := global.Get("document").Call("createElement", "a")
	blob := global.Get("Blob").New(
		[]interface{}{jsData},
		map[string]interface{}{"type": "archive/zip"},
	)
	a.Set("href", global.Get("URL").Call("createObjectURL", blob))
	a.Set("download", dir+".zip")
	a.Call("click")

	return nil
}
