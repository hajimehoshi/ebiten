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
// +build !js

package graphicscommand

import (
	"image"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
	// Screen image cannot be dumped.
	if i.screen {
		return nil
	}

	path = strings.ReplaceAll(path, "*", strconv.Itoa(i.id))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := i.dumpTo(f, graphicsDriver, blackbg, rect); err != nil {
		return err
	}

	return nil
}
