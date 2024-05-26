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

// https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/

//go:embed fxc/*.fxc
var fxcs embed.FS

func registerPrecompiledShaders() error {
	srcs := shaderprecomp.AppendBuildinShaderSources(nil)
	srcs = append(srcs, defaultShaderSourceBytes)

	for _, src := range srcs {
		// Calculate the hash of the source code to identify the Metal library.
		// FNV is used as it is fast and the hash does not need to be secure.
		h := fnv.New32()
		_, _ = h.Write(src)
		id := hex.EncodeToString(h.Sum(nil))

		vsname := id + "_vs.fxc"
		vs, err := fxcs.ReadFile("fxc/" + vsname)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "precompiled HLSL library %s was not found. Run 'go generate' for 'fxc' directory to generate them.\n", vsname)
				continue
			}
			return err
		}

		psname := id + "_ps.fxc"
		ps, err := fxcs.ReadFile("fxc/" + psname)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "precompiled HLSL library %s was not found. Run 'go generate' for 'fxc' directory to generate them.\n", psname)
				continue
			}
			return err
		}

		shaderprecomp.RegisterFXCs(src, vs, ps)
	}

	return nil
}
