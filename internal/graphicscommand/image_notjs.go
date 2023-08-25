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

//go:build !js

package graphicscommand

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) (string, error) {
	p, err := availableFilename(path)
	if err != nil {
		return "", err
	}
	path = p

	path = i.dumpName(path)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	w := bufio.NewWriter(f)
	if err := i.dumpTo(w, graphicsDriver, blackbg, rect); err != nil {
		return "", err
	}
	if err := w.Flush(); err != nil {
		return "", err
	}

	return path, nil
}

// DumpImages dumps all the current images to the specified directory.
//
// This is for testing usage.
func DumpImages(images []*Image, graphicsDriver graphicsdriver.Graphics, dir string) (string, error) {
	d, err := availableFilename(dir)
	if err != nil {
		return "", err
	}
	dir = d

	if err := os.Mkdir(dir, 0755); err != nil {
		return "", err
	}

	for _, img := range images {
		// Screen image cannot be dumped.
		if img.screen {
			continue
		}

		f, err := os.Create(filepath.Join(dir, img.dumpName("*.png")))
		if err != nil {
			return "", err
		}
		defer func() {
			_ = f.Close()
		}()

		w := bufio.NewWriter(f)
		if err := img.dumpTo(w, graphicsDriver, false, image.Rect(0, 0, img.width, img.height)); err != nil {
			return "", err
		}
		if err := w.Flush(); err != nil {
			return "", err
		}
	}

	return dir, nil
}

// availableFilename returns a filename that is valid as a new file or directory.
func availableFilename(name string) (string, error) {
	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]

	for i := 1; ; i++ {
		if _, err := os.Stat(name); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			return "", err
		}
		name = fmt.Sprintf("%s_%d%s", base, i, ext)
	}
	return name, nil
}
