// Copyright 2026 The Ebitengine Authors
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

// This is a program to precompile the shaders used by this example. It writes
// the compiled shaders into the shaders directory, where register.go picks them
// up with //go:embed.
//
// It collects the shaders with the shadercollector tool, then compiles each
// target whose compiler is available: HLSL to DXBC with fxc.exe, and MSL to a
// Metal library with the Metal tools. GLSL needs no external tool and is always
// generated.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// shader mirrors the JSON reported by the shadercollector tool.
// Only the fields used here are listed.
type shader struct {
	SourceID string
	GLSL     *struct {
		Vertex   string
		Fragment string
	}
	GLSLES *struct {
		Vertex   string
		Fragment string
	}
	HLSL *struct {
		Vertex string
		Pixel  string
	}
	MSL *struct {
		Shader string
	}
}

func main() {
	if err := xmain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// shaderDir is where the compiled shaders are written.
const shaderDir = "shaders"

func xmain() error {
	shaders, err := collectShaders()
	if err != nil {
		return err
	}

	if err := cleanDir(shaderDir); err != nil {
		return err
	}

	// GLSL is portable text and needs no external tool, so it is always generated.
	if err := generateGLSL(shaders); err != nil {
		return err
	}

	// Compile the binary targets whose compiler is available.
	if fxc := findFXC(); fxc != "" {
		if err := generateDXBC(shaders, fxc); err != nil {
			return err
		}
	} else if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "warning: the FXC shader compiler ('fxc.exe') was not found; skipping DXBC shaders.")
		fmt.Fprintln(os.Stderr, "Install the Windows SDK, then run from a Developer Command Prompt or add its bin directory to PATH, e.g.:")
		fmt.Fprintf(os.Stderr, "    C:\\Program Files (x86)\\Windows Kits\\10\\bin\\<version>\\%s\n", windowsSDKArch())
	}
	if hasMetalCompiler() {
		if err := generateMetalLibrary(shaders); err != nil {
			return err
		}
	} else if runtime.GOOS == "darwin" {
		fmt.Fprintln(os.Stderr, "warning: the Metal shader compiler ('xcrun metal') was not found; skipping Metal libraries.")
		fmt.Fprintln(os.Stderr, "Install Xcode (the Command Line Tools alone are not enough) and select it:")
		fmt.Fprintln(os.Stderr, "    sudo xcode-select -s /Applications/Xcode.app/Contents/Developer")
		fmt.Fprintln(os.Stderr, "On Xcode 16.3 or later, also download the Metal toolchain:")
		fmt.Fprintln(os.Stderr, "    xcodebuild -downloadComponent MetalToolchain")
	}

	return nil
}

// windowsSDKArch maps the host architecture to the Windows SDK bin subdirectory name.
func windowsSDKArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	case "arm":
		return "arm"
	default:
		return "x64"
	}
}

// findFXC returns the path to the fxc.exe shader compiler, or an empty string if
// it cannot be found. It first checks PATH (e.g. within a Developer Command
// Prompt), then the default Windows SDK installation directory.
func findFXC() string {
	if p, err := exec.LookPath("fxc.exe"); err == nil {
		return p
	}
	if runtime.GOOS != "windows" {
		return ""
	}

	// The SDK is installed under %ProgramFiles(x86)%\Windows Kits\10 by default.
	root := os.Getenv("ProgramFiles(x86)")
	if root == "" {
		root = `C:\Program Files (x86)`
	}
	binDir := filepath.Join(root, "Windows Kits", "10", "bin")

	entries, err := os.ReadDir(binDir)
	if err != nil {
		return ""
	}

	// Each SDK version has its own subdirectory; prefer the newest.
	var versions []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "10.") {
			versions = append(versions, e.Name())
		}
	}
	// Sort newest first. Compare the dotted versions numerically, as plain
	// string comparison would misorder, e.g., "10.0.9.0" and "10.0.22000.0".
	slices.SortFunc(versions, func(a, b string) int {
		as := strings.Split(a, ".")
		bs := strings.Split(b, ".")
		for k := range min(len(as), len(bs)) {
			ai, _ := strconv.Atoi(as[k])
			bi, _ := strconv.Atoi(bs[k])
			if ai != bi {
				return bi - ai
			}
		}
		return len(bs) - len(as)
	})

	arch := windowsSDKArch()
	for _, v := range versions {
		p := filepath.Join(binDir, v, arch, "fxc.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// hasMetalCompiler reports whether the Metal shader compiler is available.
func hasMetalCompiler() bool {
	// The metal tool is invoked through xcrun and is not on PATH directly, so
	// look it up with 'xcrun -f'.
	return exec.Command("xcrun", "-f", "metal").Run() == nil
}

// collectShaders runs the shadercollector tool and returns the reported shaders.
func collectShaders() ([]shader, error) {
	cmd := exec.Command("go", "run",
		"github.com/hajimehoshi/ebiten/v2/internal/shadercollector",
		"-target", "glsl,hlsl,msl",
		"-manifest", "manifest.json",
		".")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running shadercollector failed: %w", err)
	}

	var shaders []shader
	if err := json.Unmarshal(out, &shaders); err != nil {
		return nil, err
	}
	return shaders, nil
}

func generateGLSL(shaders []shader) error {
	for _, s := range shaders {
		if s.GLSL != nil {
			if err := writeArtifact(s.SourceID+"_vertex.glsl", []byte(s.GLSL.Vertex)); err != nil {
				return err
			}
			if err := writeArtifact(s.SourceID+"_fragment.glsl", []byte(s.GLSL.Fragment)); err != nil {
				return err
			}
		}
		if s.GLSLES != nil {
			if err := writeArtifact(s.SourceID+"_es_vertex.glsl", []byte(s.GLSLES.Vertex)); err != nil {
				return err
			}
			if err := writeArtifact(s.SourceID+"_es_fragment.glsl", []byte(s.GLSLES.Fragment)); err != nil {
				return err
			}
		}
	}
	return nil
}

func generateDXBC(shaders []shader, fxc string) error {
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	for _, s := range shaders {
		if s.HLSL == nil {
			continue
		}
		vs, err := compileDXBC(fxc, tmpdir, s.SourceID+"_vertex", s.HLSL.Vertex, "vs_4_0", "VSMain")
		if err != nil {
			return err
		}
		if err := writeArtifact(s.SourceID+"_vertex.dxbc", vs); err != nil {
			return err
		}
		ps, err := compileDXBC(fxc, tmpdir, s.SourceID+"_pixel", s.HLSL.Pixel, "ps_4_0", "PSMain")
		if err != nil {
			return err
		}
		if err := writeArtifact(s.SourceID+"_pixel.dxbc", ps); err != nil {
			return err
		}
	}
	return nil
}

// compileDXBC compiles a single HLSL source to a DXBC binary with fxc.exe.
//
// See https://learn.microsoft.com/en-us/windows/win32/direct3dtools/fxc.
func compileDXBC(fxc, tmpdir, name, source, profile, entryPoint string) ([]byte, error) {
	hlslPath := filepath.Join(tmpdir, name+".hlsl")
	if err := os.WriteFile(hlslPath, []byte(source), 0644); err != nil {
		return nil, err
	}
	outPath := filepath.Join(tmpdir, name+".dxbc")
	cmd := exec.Command(fxc, "/nologo", "/O3", "/T", profile, "/E", entryPoint, "/Fo", outPath, hlslPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return os.ReadFile(outPath)
}

// metalLibraryTargets are the platforms a Metal library is built for. A .metallib is specific to the
// SDK it was built with, so macOS and iOS need separate libraries. The suffix is appended to the
// artifact name so register.go can tell them apart. The iOS Simulator (xcrun -sdk iphonesimulator)
// is not built here, as it is not supported by the precompilation registry.
var metalLibraryTargets = []struct {
	sdk    string
	suffix string
}{
	{sdk: "macosx", suffix: "macos"},
	{sdk: "iphoneos", suffix: "ios"},
}

func generateMetalLibrary(shaders []shader) error {
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	for _, t := range metalLibraryTargets {
		// A machine might not have every SDK installed (e.g. Command Line Tools only). Skip a
		// target whose SDK is unavailable rather than failing.
		if !hasMetalSDK(t.sdk) {
			continue
		}
		for _, s := range shaders {
			if s.MSL == nil {
				continue
			}
			lib, err := compileMetalLibrary(tmpdir, s.SourceID+"_"+t.suffix, s.MSL.Shader, t.sdk)
			if err != nil {
				return err
			}
			if err := writeArtifact(s.SourceID+"_"+t.suffix+".metallib", lib); err != nil {
				return err
			}
		}
	}
	return nil
}

// hasMetalSDK reports whether the Metal shader compiler is available for the given SDK.
func hasMetalSDK(sdk string) bool {
	return exec.Command("xcrun", "-sdk", sdk, "-f", "metal").Run() == nil
}

// compileMetalLibrary compiles a single MSL source to a Metal library with the Metal tools for the
// given SDK (e.g. macosx or iphoneos).
//
// See https://developer.apple.com/documentation/metal/shader_libraries/building_a_shader_library_by_precompiling_source_files.
func compileMetalLibrary(tmpdir, name, source, sdk string) ([]byte, error) {
	metalPath := filepath.Join(tmpdir, name+".metal")
	if err := os.WriteFile(metalPath, []byte(source), 0644); err != nil {
		return nil, err
	}
	irPath := filepath.Join(tmpdir, name+".ir")
	cmd := exec.Command("xcrun", "-sdk", sdk, "metal", "-o", irPath, "-c", metalPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	libPath := filepath.Join(tmpdir, name+".metallib")
	cmd = exec.Command("xcrun", "-sdk", sdk, "metallib", "-o", libPath, irPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return os.ReadFile(libPath)
}

// writeArtifact writes data to a file named name in the shader directory.
func writeArtifact(name string, data []byte) error {
	return os.WriteFile(filepath.Join(shaderDir, name), data, 0644)
}

// cleanDir removes all generated files in dir, keeping the committed
// .gitignore and dummy.txt placeholder.
func cleanDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		switch entry.Name() {
		case ".gitignore", "dummy.txt":
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
