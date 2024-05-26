// Copyright 2024 The Ebitengine Authors
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

package main

import (
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"

	"github.com/hajimehoshi/ebiten/v2/shaderprecomp"
)

//go:embed metallib/*.metallib
var metallibs embed.FS

func registerPrecompiledShaders() error {
	srcs := shaderprecomp.AppendBuildinShaderSources(nil)
	srcs = append(srcs, defaultShaderSourceBytes)

	for _, src := range srcs {
		// Calculate the hash of the source code to identify the Metal library.
		// FNV is used as it is fast and the hash does not need to be secure.
		h := fnv.New32()
		_, _ = h.Write(src)
		id := hex.EncodeToString(h.Sum(nil))

		name := id + ".metallib"
		lib, err := metallibs.ReadFile("metallib/" + name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "precompiled Metal library %s was not found. Run 'go generate' for 'metallib' directory to generate them.\n", name)
				continue
			}
			return err
		}
		shaderprecomp.RegisterMetalLibrary(src, lib)
	}

	return nil
}
