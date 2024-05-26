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

// This is a program to generate precompiled HLSL blobs (FXC files).
//
// See https://learn.microsoft.com/en-us/windows/win32/direct3dtools/fxc.
package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
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
	if _, err := exec.LookPath("fxc.exe"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "fxc.exe not found. Please install Windows SDK.")
			fmt.Fprintln(os.Stderr, "See https://learn.microsoft.com/en-us/windows/win32/direct3dtools/fxc for more details.")
			fmt.Fprintln(os.Stderr, "On PowerShell, you can add a path to the PATH environment variable temporarily like:")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, `    & (Get-Process -Id $PID).Path { $env:PATH="C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64;"+$env:PATH; go generate .\examples\shaderprecomp\fxc\ }`)
			fmt.Fprintln(os.Stderr)
			os.Exit(1)
		}
		return err
	}

	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	srcs := shaderprecomp.AppendBuildinShaderSources(nil)

	defaultSrc, err := os.ReadFile(filepath.Join("..", "defaultshader.go"))
	if err != nil {
		return err
	}
	srcs = append(srcs, defaultSrc)

	for _, src := range srcs {
		// Avoid using errgroup.Group.
		// Compiling sources in parallel causes a mixed error message on the console.
		if err := compile(src, tmpdir); err != nil {
			return err
		}
	}
	return nil
}

func generateHSLSFiles(source []byte, id string, tmpdir string) (vs, ps string, err error) {
	vsHLSLFilePath := filepath.Join(tmpdir, id+"_vs.hlsl")
	vsf, err := os.Create(vsHLSLFilePath)
	if err != nil {
		return "", "", err
	}
	defer vsf.Close()

	psHLSLFilePath := filepath.Join(tmpdir, id+"_ps.hlsl")
	psf, err := os.Create(psHLSLFilePath)
	if err != nil {
		return "", "", err
	}
	defer psf.Close()

	if err := shaderprecomp.CompileToHLSL(vsf, psf, source); err != nil {
		return "", "", err
	}

	return vsHLSLFilePath, psHLSLFilePath, nil
}

func compile(kageSource []byte, tmpdir string) error {
	h := fnv.New32()
	_, _ = h.Write(kageSource)
	id := hex.EncodeToString(h.Sum(nil))

	// Generate HLSL files. Make sure this process doesn't have any handlers of the files.
	// Without closing the files, fxc.exe cannot access the files.
	vsHLSLFilePath, psHLSLFilePath, err := generateHSLSFiles(kageSource, id, tmpdir)
	if err != nil {
		return err
	}

	vsFXCFilePath := id + "_vs.fxc"
	cmd := exec.Command("fxc.exe", "/nologo", "/O3", "/T", shaderprecomp.HLSLVertexShaderProfile, "/E", shaderprecomp.HLSLVertexShaderEntryPoint, "/Fo", vsFXCFilePath, vsHLSLFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	psFXCFilePath := id + "_ps.fxc"
	cmd = exec.Command("fxc.exe", "/nologo", "/O3", "/T", shaderprecomp.HLSLPixelShaderProfile, "/E", shaderprecomp.HLSLPixelShaderEntryPoint, "/Fo", psFXCFilePath, psHLSLFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
