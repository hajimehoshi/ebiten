// Copyright 2022 The Ebitengine Authors
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

package graphicscommand

import (
	"archive/zip"
	"bytes"
	"image"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func download(buf *bytes.Buffer, mime string, path string) {
	global := js.Global()

	jsData := global.Get("Uint8Array").New(buf.Len())
	js.CopyBytesToJS(jsData, buf.Bytes())

	a := global.Get("document").Call("createElement", "a")
	blob := global.Get("Blob").New(
		[]any{jsData},
		map[string]any{"type": mime},
	)
	a.Set("href", global.Get("URL").Call("createObjectURL", blob))
	a.Set("download", path)
	a.Call("click")
}

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) (string, error) {
	// Screen image cannot be dumped.
	if i.screen {
		return "", nil
	}

	buf := &bytes.Buffer{}
	if err := i.dumpTo(buf, graphicsDriver, blackbg, rect); err != nil {
		return "", err
	}

	download(buf, "image/png", i.dumpName(path))

	return path, nil
}

// DumpImages dumps all the specified images to the specified directory.
//
// This is for testing usage.
func DumpImages(images []*Image, graphicsDriver graphicsdriver.Graphics, dir string) (string, error) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	for _, img := range images {
		// Screen image cannot be dumped.
		if img.screen {
			continue
		}

		f, err := zw.Create(img.dumpName("*.png"))
		if err != nil {
			return "", err
		}

		if err := img.dumpTo(f, graphicsDriver, false, image.Rect(0, 0, img.width, img.height)); err != nil {
			return "", err
		}
	}

	_ = zw.Close()

	zip := dir + ".zip"
	download(buf, "archive/zip", zip)

	return zip, nil
}
