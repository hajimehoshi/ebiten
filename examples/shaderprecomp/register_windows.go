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
	"fmt"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/shaderprecomp"
)

// https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/

//go:embed fxc/*.fxc
var fxcs embed.FS

func registerPrecompiledShaders() error {
	ents, err := fxcs.ReadDir("fxc")
	if err != nil {
		return err
	}

	var registered bool
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}

		const suffix = "_vs.fxc"
		name := ent.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}

		id := name[:len(name)-len(suffix)]
		srcID, err := shaderprecomp.ParseSourceID(id)
		if err != nil {
			continue
		}

		vs, err := fxcs.ReadFile("fxc/" + id + "_vs.fxc")
		if err != nil {
			return err
		}
		ps, err := fxcs.ReadFile("fxc/" + id + "_ps.fxc")
		if err != nil {
			return err
		}

		shaderprecomp.RegisterFXCs(srcID, vs, ps)
		registered = true
	}

	if !registered {
		fmt.Fprintln(os.Stderr, "precompiled HLSL libraries were not found. Run 'go generate' for 'fxc' directory to generate them.")
	}

	return nil
}
