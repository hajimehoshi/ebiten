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

// Package shaderprecomp offers APIs to register precompiled shaders to reduce runtime shader
// compilation work and shorten loading time.
//
// This package is experimental and the API might change in the future.
//
// # Collecting shader sources
//
// The shadercollector tool collects explicitly marked Kage sources from Go packages and their
// dependencies. To mark shader files, add a directive outside any function in a Go source file:
//
//	//ebitengine:shaderfile shaders/*.kage
//
// Paths are relative to the package directory. To mark a package-level string constant instead,
// place //ebitengine:shadersource directly above its declaration. A //go:embed directive alone
// does not make a shader discoverable by shadercollector.
//
// Alternatively, list shader files in a JSON manifest. Paths are relative to the manifest file:
//
//	{"ShaderFiles": ["shaders/effect.kage"]}
//
// Run shadercollector from the application's module with the desired targets:
//
//	go run github.com/hajimehoshi/ebiten/v2/internal/shadercollector -target glsl,hlsl,msl -manifest manifest.json ./...
//
// Omit -manifest when using only directives. The tool emits a JSON array containing each shader's
// SourceID and converted backend sources. The command above requests glsl, hlsl, and msl;
// glsl emits both desktop GLSL and GLSL ES. Package dependencies include Ebitengine's marked
// built-in shaders, so collect packages as well as any files listed in a manifest.
//
// # Compiling backend artifacts
//
// The collector emits source code, not platform binaries. Prepare the artifacts required by
// each registration function:
//
//   - For [RegisterGLSL], use the GLSL and GLSL ES vertex and fragment sources directly.
//   - For [RegisterDXBCsForWindows], compile the HLSL vertex and pixel sources with the Windows
//     SDK fxc tool. The example uses entry points VSMain and PSMain with profiles vs_4_0 and ps_4_0.
//   - For [RegisterDXBCsForXbox], compile HLSL with the Xbox GDK shader compiler.
//   - For [RegisterMetalLibraryForMacOS] and [RegisterMetalLibraryForIOS], compile MSL into
//     .metallib files with xcrun metal and xcrun metallib, using the macosx and iphoneos SDKs
//     respectively. Each platform requires a separate library.
//
// # Registering and using shaders
//
// Embed or load the artifacts and register them during application initialization, before using
// the shaders. Pass each reported SourceID to [ParseShaderSourceID] or [MustParseShaderSourceID],
// then call the matching registration function with that ID and its artifact bytes. For example,
// given a collector-reported sourceID and an embedded macOS library:
//
//	id := shaderprecomp.MustParseShaderSourceID(sourceID)
//	shaderprecomp.RegisterMetalLibraryForMacOS(id, library)
//
// Register each shader only once for each artifact category; duplicate registrations panic.
// Keep the original Kage source and continue passing it to ebiten.NewShader. Drawing and uniform
// APIs are unchanged. Ebitengine selects the registered artifacts matching the shader source ID
// and the active backend and platform.
//
// Precompilation does not eliminate all runtime compilation: ebiten.NewShader still compiles
// Kage into an intermediate representation. GLSL registration skips conversion to GLSL, but
// OpenGL still compiles and links the GLSL sources at runtime.
//
// On OpenGL, DirectX, and Metal, missing artifacts fall back to runtime compilation.
// The iOS Simulator does not support registered Metal libraries and uses
// runtime compilation. For GLSL, an omitted flavor falls back to runtime source conversion.
//
// Regenerate artifacts after changing shader sources or updating Ebitengine. A [ShaderSourceID]
// is not stable across Ebitengine versions; use the application's Ebitengine version for collection
// and precompilation.
//
// # Example
//
// The [shaderprecomp example] includes a generator and registration code using go:embed.
// From the Ebitengine repository root, run:
//
//	go generate ./examples/shaderprecomp
//	go run ./examples/shaderprecomp
//
// The generator always produces GLSL and additionally compiles DXBC and Metal libraries when
// their compilers and SDKs are available. Windows DXBC requires the Windows SDK fxc tool;
// Metal libraries require Xcode's Metal toolchain and the corresponding platform SDKs.
//
// [shaderprecomp example]: https://github.com/hajimehoshi/ebiten/tree/main/examples/shaderprecomp
package shaderprecomp
