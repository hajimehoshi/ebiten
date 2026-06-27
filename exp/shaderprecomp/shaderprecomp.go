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

// Package shaderprecomp offers APIs to register precompiled shaders.
//
// Registering a precompiled shader lets Ebitengine skip compiling the shader at runtime,
// which shortens the loading time.
//
// A precompiled shader is keyed by a [ShaderSourceID]. The shadercollector tool reports the
// shader sources used by the given packages, each with a "SourceID" and, with the -target option,
// their converted sources for the targets (a comma-separated list of glsl, hlsl, msl, and pssl):
//
//	go run github.com/hajimehoshi/ebiten/v2/internal/shadercollector -target hlsl,msl ./...
//
// Pass a reported "SourceID" to [ParseShaderSourceID] or [MustParseShaderSourceID] to get a
// [ShaderSourceID], then register the precompiled shaders for that ID with the Register functions.
//
// This package is experimental and the API might change in the future.
package shaderprecomp

import (
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	internalshaderprecomp "github.com/hajimehoshi/ebiten/v2/internal/shaderprecomp"
)

// ShaderSourceID is an identifier of a shader source.
//
// The shadercollector tool reports a ShaderSourceID as a base32 string.
// Use [ParseShaderSourceID] or [MustParseShaderSourceID] to reconstruct it.
//
// A ShaderSourceID is not stable across Ebitengine versions: the same shader source might be
// assigned a different ID when Ebitengine is updated. Precompile shaders for the specific
// Ebitengine version you target, and do not expect an ID obtained from one version to be valid
// in another.
type ShaderSourceID struct {
	id shaderir.SourceID
}

// String returns the base32 string representation of the ShaderSourceID,
// matching the value reported by the shadercollector tool.
func (id ShaderSourceID) String() string {
	return id.id.String()
}

// ParseShaderSourceID parses a base32 string representation of a [ShaderSourceID],
// as reported by the shadercollector tool or by [ShaderSourceID.String].
func ParseShaderSourceID(str string) (ShaderSourceID, error) {
	id, err := shaderir.ParseSourceID(str)
	if err != nil {
		return ShaderSourceID{}, err
	}
	return ShaderSourceID{id: id}, nil
}

// MustParseShaderSourceID is like [ParseShaderSourceID] but panics if str is invalid.
// MustParseShaderSourceID is useful in generated code, where str is always a valid value.
func MustParseShaderSourceID(str string) ShaderSourceID {
	id, err := ParseShaderSourceID(str)
	if err != nil {
		panic(err)
	}
	return id
}

// RegisterFXCsForWindows registers precompiled FXC binaries for Windows for the shader source
// identified by id, so that Ebitengine uses them on DirectX instead of compiling the shader at runtime.
//
// vertexFXC and pixelFXC are the binaries compiled by the Windows SDK fxc tool.
//
// Windows and Xbox are registered separately as a precaution: Xbox uses the GDK's own shader compiler,
// and whether a binary built for one platform is usable on the other is not guaranteed. If they turn out
// to be interchangeable for your shaders, the same binaries can be registered for both.
//
// RegisterFXCsForWindows panics if Windows FXCs for id are already registered.
//
// RegisterFXCsForWindows is concurrent-safe.
func RegisterFXCsForWindows(id ShaderSourceID, vertexFXC, pixelFXC []byte) {
	internalshaderprecomp.RegisterFXCsForWindows(id.id, vertexFXC, pixelFXC)
}

// RegisterFXCsForXbox registers precompiled FXC binaries for Xbox for the shader source identified
// by id, so that Ebitengine uses them on DirectX instead of compiling the shader at runtime.
//
// vertexFXC and pixelFXC are the binaries compiled by the Xbox (GDK) shader compiler.
// See [RegisterFXCsForWindows] for why Windows and Xbox are registered separately.
//
// RegisterFXCsForXbox panics if Xbox FXCs for id are already registered.
//
// RegisterFXCsForXbox is concurrent-safe.
func RegisterFXCsForXbox(id ShaderSourceID, vertexFXC, pixelFXC []byte) {
	internalshaderprecomp.RegisterFXCsForXbox(id.id, vertexFXC, pixelFXC)
}

// RegisterMetalLibraryForMacOS registers a precompiled macOS Metal library for the shader source
// identified by id, so that Ebitengine uses it on macOS instead of compiling the shader at runtime.
//
// library is the content of a .metallib file built with the macOS SDK (xcrun -sdk macosx). A library
// built for a different platform cannot be loaded, so a macOS build and an iOS build need separate
// libraries; register each with the matching function.
//
// RegisterMetalLibraryForMacOS panics if a macOS Metal library for id is already registered.
//
// RegisterMetalLibraryForMacOS is concurrent-safe.
func RegisterMetalLibraryForMacOS(id ShaderSourceID, library []byte) {
	internalshaderprecomp.RegisterMetalLibraryForMacOS(id.id, library)
}

// RegisterMetalLibraryForIOS registers a precompiled iOS Metal library for the shader source identified
// by id, so that Ebitengine uses it on iOS instead of compiling the shader at runtime.
//
// library is the content of a .metallib file built with the iOS device SDK (xcrun -sdk iphoneos).
//
// The iOS Simulator needs a library built with a different SDK (xcrun -sdk iphonesimulator) and is not
// supported; Ebitengine compiles the shader at runtime there.
//
// RegisterMetalLibraryForIOS panics if an iOS Metal library for id is already registered.
//
// RegisterMetalLibraryForIOS is concurrent-safe.
func RegisterMetalLibraryForIOS(id ShaderSourceID, library []byte) {
	internalshaderprecomp.RegisterMetalLibraryForIOS(id.id, library)
}

// RegisterPlayStation5Shader registers a precompiled PlayStation 5 shader for the shader source
// identified by id, so that Ebitengine uses it instead of compiling the shader at runtime.
//
// RegisterPlayStation5Shader panics if a PlayStation 5 shader for id is already registered.
//
// RegisterPlayStation5Shader is concurrent-safe.
func RegisterPlayStation5Shader(id ShaderSourceID, vertexHeader, vertexText, pixelHeader, pixelText []byte) {
	internalshaderprecomp.RegisterPlayStation5Shader(id.id, vertexHeader, vertexText, pixelHeader, pixelText)
}

// RegisterGLSL registers precompiled GLSL shaders for the shader source identified by id,
// so that Ebitengine uses them on OpenGL instead of converting the shader source at runtime.
//
// vertex and fragment are the GLSL shaders, and esVertex and esFragment are the GLSL ES shaders.
// Which flavor is used is determined at runtime, because the OpenGL version is not known until then,
// so both flavors should usually be registered. A flavor may be nil, in which case Ebitengine falls
// back to converting the shader source at runtime when that flavor is needed.
//
// RegisterGLSL panics if GLSL shaders for id are already registered.
//
// RegisterGLSL is concurrent-safe.
func RegisterGLSL(id ShaderSourceID, vertex, fragment, esVertex, esFragment []byte) {
	internalshaderprecomp.RegisterGLSL(id.id, vertex, fragment, esVertex, esFragment)
}
