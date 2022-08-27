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
	"bytes"
	"image"
	"path/filepath"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
	// Screen image cannot be dumped.
	if i.screen {
		return nil
	}

	path = strings.ReplaceAll(path, "*", strconv.Itoa(i.id))
	path = filepath.Base(path)

	global := js.Global()

	buf := &bytes.Buffer{}
	if err := i.dumpTo(buf, graphicsDriver, blackbg, rect); err != nil {
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

