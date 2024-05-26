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

//go:build ignore

// This is a program to generate precompiled Metal libraries.
//
// See https://developer.apple.com/documentation/metal/shader_libraries/building_a_shader_library_by_precompiling_source_files.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2/shaderprecomp"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	srcs := shaderprecomp.AppendBuildinShaderSources(nil)

	defaultSrcBytes, err := os.ReadFile(filepath.Join("..", "defaultshader.go"))
	if err != nil {
		return err
	}
	srcs = append(srcs, shaderprecomp.NewShaderSource(defaultSrcBytes))

	for i, src := range srcs {
		// Avoid using errgroup.Group.
		// Compiling sources in parallel causes a mixed error message on the console.
		if err := compile(src, i, tmpdir); err != nil {
			return err
		}
	}
	return nil
}

func compile(source *shaderprecomp.ShaderSource, index int, tmpdir string) error {
	metalFilePath := filepath.Join(tmpdir, fmt.Sprintf("%d.metal", index))

	f, err := os.Create(metalFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := shaderprecomp.CompileToMSL(f, source); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}

	irFilePath := filepath.Join(tmpdir, fmt.Sprintf("%d.ir", index))
	cmd := exec.Command("xcrun", "-sdk", "macosx", "metal", "-o", irFilePath, "-c", metalFilePath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	metallibFilePath := fmt.Sprintf("%d.metallib", index)
	cmd = exec.Command("xcrun", "-sdk", "macosx", "metallib", "-o", metallibFilePath, irFilePath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
