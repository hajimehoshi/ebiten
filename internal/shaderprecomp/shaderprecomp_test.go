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

package shaderprecomp_test

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderprecomp"
)

func id(s string) shaderir.SourceID {
	return shaderir.CalcSourceID([]byte(s))
}

func TestFXCs(t *testing.T) {
	i := id("fxc")
	if _, _, ok := shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformWindows); ok {
		t.Fatal("Windows FXCs must not be registered yet")
	}
	if _, _, ok := shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformXbox); ok {
		t.Fatal("Xbox FXCs must not be registered yet")
	}

	shaderprecomp.RegisterFXCsForWindows(i, []byte("winvs"), []byte("winps"))
	shaderprecomp.RegisterFXCsForXbox(i, []byte("xboxvs"), []byte("xboxps"))

	vs, ps, ok := shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformWindows)
	if !ok {
		t.Fatal("Windows FXCs must be registered")
	}
	if got, want := string(vs)+"/"+string(ps), "winvs/winps"; got != want {
		t.Errorf("Windows FXCs: got %q, want %q", got, want)
	}

	vs, ps, ok = shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformXbox)
	if !ok {
		t.Fatal("Xbox FXCs must be registered")
	}
	if got, want := string(vs)+"/"+string(ps), "xboxvs/xboxps"; got != want {
		t.Errorf("Xbox FXCs: got %q, want %q", got, want)
	}
}

func TestFXCsWindowsOnly(t *testing.T) {
	i := id("fxc-windows-only")
	shaderprecomp.RegisterFXCsForWindows(i, []byte("vs"), []byte("ps"))

	if _, _, ok := shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformWindows); !ok {
		t.Error("the Windows FXCs must be available")
	}
	if _, _, ok := shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformXbox); ok {
		t.Error("the Xbox FXCs must not be available when not registered")
	}
}

func TestFXCsUnregistered(t *testing.T) {
	i := id("fxc-unregistered")
	if _, _, ok := shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformWindows); ok {
		t.Error("the Windows FXCs must not be available for an unregistered ID")
	}
	if _, _, ok := shaderprecomp.FXCs(i, shaderprecomp.FXCPlatformXbox); ok {
		t.Error("the Xbox FXCs must not be available for an unregistered ID")
	}
}

func TestMetalLibrary(t *testing.T) {
	i := id("metal")
	if _, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformMacOS); ok {
		t.Fatal("a macOS Metal library must not be registered yet")
	}
	if _, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformIOS); ok {
		t.Fatal("an iOS Metal library must not be registered yet")
	}

	shaderprecomp.RegisterMetalLibraryForMacOS(i, []byte("maclib"))
	shaderprecomp.RegisterMetalLibraryForIOS(i, []byte("ioslib"))

	lib, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformMacOS)
	if !ok {
		t.Fatal("a macOS Metal library must be registered")
	}
	if got, want := string(lib), "maclib"; got != want {
		t.Errorf("macOS library: got %q, want %q", got, want)
	}

	lib, ok = shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformIOS)
	if !ok {
		t.Fatal("an iOS Metal library must be registered")
	}
	if got, want := string(lib), "ioslib"; got != want {
		t.Errorf("iOS library: got %q, want %q", got, want)
	}

	// The iOS Simulator has no precompiled library and falls back to runtime compilation.
	if _, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformIOSSimulator); ok {
		t.Error("the iOS Simulator must not use a precompiled Metal library")
	}
}

func TestMetalLibraryMacOSOnly(t *testing.T) {
	i := id("metal-macos-only")
	shaderprecomp.RegisterMetalLibraryForMacOS(i, []byte("maclib"))

	if _, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformMacOS); !ok {
		t.Error("the macOS Metal library must be available")
	}
	if _, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformIOS); ok {
		t.Error("the iOS Metal library must not be available when not registered")
	}
}

func TestMetalLibraryUnregistered(t *testing.T) {
	i := id("metal-unregistered")
	if _, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformMacOS); ok {
		t.Error("the macOS Metal library must not be available for an unregistered ID")
	}
	if _, ok := shaderprecomp.MetalLibrary(i, shaderprecomp.MetalLibraryPlatformIOS); ok {
		t.Error("the iOS Metal library must not be available for an unregistered ID")
	}
}

func TestRegisterMetalLibraryDuplicatePanics(t *testing.T) {
	i := id("metal-dup")
	shaderprecomp.RegisterMetalLibraryForMacOS(i, []byte("maclib"))

	defer func() {
		if recover() == nil {
			t.Error("registering a macOS Metal library twice for the same ID must panic")
		}
	}()
	shaderprecomp.RegisterMetalLibraryForMacOS(i, []byte("maclib2"))
}

func TestPlayStation5Shader(t *testing.T) {
	i := id("ps5")
	if _, _, _, _, ok := shaderprecomp.PlayStation5Shader(i); ok {
		t.Fatal("a PlayStation 5 shader must not be registered yet")
	}
	shaderprecomp.RegisterPlayStation5Shader(i, []byte("vh"), []byte("vt"), []byte("ph"), []byte("pt"))
	vh, vt, ph, pt, ok := shaderprecomp.PlayStation5Shader(i)
	if !ok {
		t.Fatal("a PlayStation 5 shader must be registered")
	}
	if got, want := string(vh), "vh"; got != want {
		t.Errorf("vertex header: got %q, want %q", got, want)
	}
	if got, want := string(vt), "vt"; got != want {
		t.Errorf("vertex text: got %q, want %q", got, want)
	}
	if got, want := string(ph), "ph"; got != want {
		t.Errorf("pixel header: got %q, want %q", got, want)
	}
	if got, want := string(pt), "pt"; got != want {
		t.Errorf("pixel text: got %q, want %q", got, want)
	}
}

func TestGLSL(t *testing.T) {
	i := id("glsl")
	shaderprecomp.RegisterGLSL(i, []byte("vs"), []byte("fs"), []byte("esvs"), []byte("esfs"))

	vs, fs, ok := shaderprecomp.GLSL(i, false)
	if !ok {
		t.Fatal("the default GLSL must be registered")
	}
	if got, want := string(vs)+"/"+string(fs), "vs/fs"; got != want {
		t.Errorf("default GLSL: got %q, want %q", got, want)
	}

	esvs, esfs, ok := shaderprecomp.GLSL(i, true)
	if !ok {
		t.Fatal("the GLSL ES must be registered")
	}
	if got, want := string(esvs)+"/"+string(esfs), "esvs/esfs"; got != want {
		t.Errorf("GLSL ES: got %q, want %q", got, want)
	}
}

func TestGLSLESOnly(t *testing.T) {
	i := id("glsl-es-only")
	shaderprecomp.RegisterGLSL(i, nil, nil, []byte("esvs"), []byte("esfs"))

	if _, _, ok := shaderprecomp.GLSL(i, false); ok {
		t.Error("the default GLSL must not be available when registered as nil")
	}
	if _, _, ok := shaderprecomp.GLSL(i, true); !ok {
		t.Error("the GLSL ES must be available")
	}
}

func TestGLSLUnregistered(t *testing.T) {
	i := id("glsl-unregistered")
	if _, _, ok := shaderprecomp.GLSL(i, false); ok {
		t.Error("GLSL must not be available for an unregistered ID")
	}
	if _, _, ok := shaderprecomp.GLSL(i, true); ok {
		t.Error("GLSL ES must not be available for an unregistered ID")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	i := id("dup")
	shaderprecomp.RegisterFXCsForWindows(i, []byte("vs"), []byte("ps"))

	defer func() {
		if recover() == nil {
			t.Error("registering Windows FXCs twice for the same ID must panic")
		}
	}()
	shaderprecomp.RegisterFXCsForWindows(i, []byte("vs2"), []byte("ps2"))
}

func TestKeyedBySourceID(t *testing.T) {
	// Different sources must yield different keys.
	a := id("source-a")
	b := id("source-b")
	if bytes.Equal(a[:], b[:]) {
		t.Fatal("distinct sources must have distinct IDs")
	}
	shaderprecomp.RegisterMetalLibraryForMacOS(a, []byte("a"))
	shaderprecomp.RegisterMetalLibraryForMacOS(b, []byte("b"))
	if lib, _ := shaderprecomp.MetalLibrary(a, shaderprecomp.MetalLibraryPlatformMacOS); string(lib) != "a" {
		t.Errorf("got %q, want %q", lib, "a")
	}
	if lib, _ := shaderprecomp.MetalLibrary(b, shaderprecomp.MetalLibraryPlatformMacOS); string(lib) != "b" {
		t.Errorf("got %q, want %q", lib, "b")
	}
}
