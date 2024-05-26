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
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/hajimehoshi/ebiten/v2/shaderprecomp"
)

//go:embed metallib/*.metallib
var metallibs embed.FS

func registerPrecompiledShaders() error {
	srcs := shaderprecomp.AppendBuildinShaderSources(nil)
	defaultShaderSource, err := shaderprecomp.NewShaderSource(defaultShaderSourceBytes)
	if err != nil {
		return err
	}
	srcs = append(srcs, defaultShaderSource)

	for _, src := range srcs {
		name := src.ID().String() + ".metallib"
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
