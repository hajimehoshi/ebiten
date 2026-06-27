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

// Package shaderprecomp is a platform-neutral registry of precompiled shaders.
//
// The public registration APIs live in the experimental package
// github.com/hajimehoshi/ebiten/v2/exp/shaderprecomp, which forwards to this registry.
// Each graphics driver reads its own kind of precompiled shader from this registry.
// Registrations for a driver that is not active on the current platform are simply never read.
package shaderprecomp

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// DXBCPlatform identifies the platform a precompiled DXBC binary targets.
//
// Windows and Xbox are kept separate as a precaution: Xbox uses the GDK's own shader compiler, and a
// binary built for one platform is not guaranteed to be usable on the other. They may turn out to be
// interchangeable, but the registry distinguishes them so a Windows binary is never silently used on Xbox.
type DXBCPlatform int

const (
	// DXBCPlatformWindows is a DXBC binary compiled with the Windows SDK fxc tool.
	DXBCPlatformWindows DXBCPlatform = iota

	// DXBCPlatformXbox is a DXBC binary compiled with the Xbox (GDK) shader compiler.
	DXBCPlatformXbox
)

type dxbcBinaries struct {
	vertex []byte
	pixel  []byte
}

type dxbc struct {
	windows dxbcBinaries
	xbox    dxbcBinaries
}

// MetalLibraryPlatform identifies the platform a precompiled Metal library targets.
//
// A .metallib is specific to the SDK it was built with, so a library built for one platform
// cannot be loaded on another.
type MetalLibraryPlatform int

const (
	// MetalLibraryPlatformMacOS is a Metal library built with the macOS SDK (xcrun -sdk macosx).
	MetalLibraryPlatformMacOS MetalLibraryPlatform = iota

	// MetalLibraryPlatformIOS is a Metal library built with the iOS device SDK (xcrun -sdk iphoneos).
	MetalLibraryPlatformIOS

	// MetalLibraryPlatformIOSSimulator is the iOS Simulator, which needs the iOS Simulator SDK
	// (xcrun -sdk iphonesimulator). No precompiled library is registered for it; Ebitengine falls
	// back to runtime compilation there.
	//
	// TODO: Support registering a precompiled Metal library for the iOS Simulator.
	MetalLibraryPlatformIOSSimulator
)

type metalLibrary struct {
	macOS []byte
	iOS   []byte
}

type playStation5Shader struct {
	vertexHeader []byte
	vertexText   []byte
	pixelHeader  []byte
	pixelText    []byte
}

type glslShader struct {
	vertex     []byte
	fragment   []byte
	esVertex   []byte
	esFragment []byte
}

// registry holds all the registered precompiled shaders.
// Registration usually happens once at initialization, so a single mutex guarding all the maps is sufficient.
type registry struct {
	mu sync.Mutex

	dxbcs               map[shaderir.SourceID]dxbc
	metalLibraries      map[shaderir.SourceID]metalLibrary
	playStation5Shaders map[shaderir.SourceID]playStation5Shader
	glslShaders         map[shaderir.SourceID]glslShader
}

var theRegistry registry

func (r *registry) registerDXBCsForWindows(id shaderir.SourceID, vertex, pixel []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.dxbcs == nil {
		r.dxbcs = map[shaderir.SourceID]dxbc{}
	}
	f := r.dxbcs[id]
	if f.windows.vertex != nil || f.windows.pixel != nil {
		panic(fmt.Sprintf("shaderprecomp: Windows DXBCs for the shader source ID %s are already registered", id))
	}
	f.windows = dxbcBinaries{vertex: vertex, pixel: pixel}
	r.dxbcs[id] = f
}

func (r *registry) registerDXBCsForXbox(id shaderir.SourceID, vertex, pixel []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.dxbcs == nil {
		r.dxbcs = map[shaderir.SourceID]dxbc{}
	}
	f := r.dxbcs[id]
	if f.xbox.vertex != nil || f.xbox.pixel != nil {
		panic(fmt.Sprintf("shaderprecomp: Xbox DXBCs for the shader source ID %s are already registered", id))
	}
	f.xbox = dxbcBinaries{vertex: vertex, pixel: pixel}
	r.dxbcs[id] = f
}

func (r *registry) getDXBCs(id shaderir.SourceID, platform DXBCPlatform) (vertex, pixel []byte, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.dxbcs[id]
	var b dxbcBinaries
	switch platform {
	case DXBCPlatformWindows:
		b = f.windows
	case DXBCPlatformXbox:
		b = f.xbox
	default:
		return nil, nil, false
	}
	return b.vertex, b.pixel, b.vertex != nil || b.pixel != nil
}

func (r *registry) registerMetalLibraryForMacOS(id shaderir.SourceID, library []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.metalLibraries == nil {
		r.metalLibraries = map[shaderir.SourceID]metalLibrary{}
	}
	lib := r.metalLibraries[id]
	if lib.macOS != nil {
		panic(fmt.Sprintf("shaderprecomp: a macOS Metal library for the shader source ID %s is already registered", id))
	}
	lib.macOS = library
	r.metalLibraries[id] = lib
}

func (r *registry) registerMetalLibraryForIOS(id shaderir.SourceID, library []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.metalLibraries == nil {
		r.metalLibraries = map[shaderir.SourceID]metalLibrary{}
	}
	lib := r.metalLibraries[id]
	if lib.iOS != nil {
		panic(fmt.Sprintf("shaderprecomp: an iOS Metal library for the shader source ID %s is already registered", id))
	}
	lib.iOS = library
	r.metalLibraries[id] = lib
}

func (r *registry) getMetalLibrary(id shaderir.SourceID, platform MetalLibraryPlatform) (library []byte, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lib := r.metalLibraries[id]
	switch platform {
	case MetalLibraryPlatformMacOS:
		return lib.macOS, lib.macOS != nil
	case MetalLibraryPlatformIOS:
		return lib.iOS, lib.iOS != nil
	default:
		// There is no precompiled Metal library for the iOS Simulator,
		// so fall back to runtime compilation.
		return nil, false
	}
}

func (r *registry) registerPlayStation5Shader(id shaderir.SourceID, vertexHeader, vertexText, pixelHeader, pixelText []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.playStation5Shaders == nil {
		r.playStation5Shaders = map[shaderir.SourceID]playStation5Shader{}
	}
	if _, ok := r.playStation5Shaders[id]; ok {
		panic(fmt.Sprintf("shaderprecomp: a PlayStation 5 shader for the shader source ID %s is already registered", id))
	}
	r.playStation5Shaders[id] = playStation5Shader{
		vertexHeader: vertexHeader,
		vertexText:   vertexText,
		pixelHeader:  pixelHeader,
		pixelText:    pixelText,
	}
}

func (r *registry) getPlayStation5Shader(id shaderir.SourceID) (vertexHeader, vertexText, pixelHeader, pixelText []byte, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.playStation5Shaders[id]
	return s.vertexHeader, s.vertexText, s.pixelHeader, s.pixelText, ok
}

func (r *registry) registerGLSL(id shaderir.SourceID, vertex, fragment, esVertex, esFragment []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.glslShaders == nil {
		r.glslShaders = map[shaderir.SourceID]glslShader{}
	}
	if _, ok := r.glslShaders[id]; ok {
		panic(fmt.Sprintf("shaderprecomp: GLSL shaders for the shader source ID %s are already registered", id))
	}
	r.glslShaders[id] = glslShader{
		vertex:     vertex,
		fragment:   fragment,
		esVertex:   esVertex,
		esFragment: esFragment,
	}
}

func (r *registry) getGLSL(id shaderir.SourceID, es bool) (vertex, fragment []byte, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	g, ok := r.glslShaders[id]
	if !ok {
		return nil, nil, false
	}
	if es {
		if g.esVertex == nil && g.esFragment == nil {
			return nil, nil, false
		}
		return g.esVertex, g.esFragment, true
	}
	if g.vertex == nil && g.fragment == nil {
		return nil, nil, false
	}
	return g.vertex, g.fragment, true
}

// RegisterDXBCsForWindows registers precompiled Windows DXBC binaries for the shader source ID id.
// RegisterDXBCsForWindows panics if Windows DXBCs for id are already registered.
func RegisterDXBCsForWindows(id shaderir.SourceID, vertex, pixel []byte) {
	theRegistry.registerDXBCsForWindows(id, vertex, pixel)
}

// RegisterDXBCsForXbox registers precompiled Xbox DXBC binaries for the shader source ID id.
// RegisterDXBCsForXbox panics if Xbox DXBCs for id are already registered.
func RegisterDXBCsForXbox(id shaderir.SourceID, vertex, pixel []byte) {
	theRegistry.registerDXBCsForXbox(id, vertex, pixel)
}

// DXBCs returns the precompiled DXBC binaries registered for the shader source ID id and platform.
// ok is false when no binaries are registered for the platform, in which case the shader is compiled at runtime.
func DXBCs(id shaderir.SourceID, platform DXBCPlatform) (vertex, pixel []byte, ok bool) {
	return theRegistry.getDXBCs(id, platform)
}

// RegisterMetalLibraryForMacOS registers a precompiled macOS Metal library for the shader source ID id.
// RegisterMetalLibraryForMacOS panics if a macOS Metal library for id is already registered.
func RegisterMetalLibraryForMacOS(id shaderir.SourceID, library []byte) {
	theRegistry.registerMetalLibraryForMacOS(id, library)
}

// RegisterMetalLibraryForIOS registers a precompiled iOS Metal library for the shader source ID id.
// RegisterMetalLibraryForIOS panics if an iOS Metal library for id is already registered.
func RegisterMetalLibraryForIOS(id shaderir.SourceID, library []byte) {
	theRegistry.registerMetalLibraryForIOS(id, library)
}

// MetalLibrary returns the precompiled Metal library registered for the shader source ID id and platform.
// ok is false when no library is registered for the platform, in which case the shader is compiled at runtime.
func MetalLibrary(id shaderir.SourceID, platform MetalLibraryPlatform) (library []byte, ok bool) {
	return theRegistry.getMetalLibrary(id, platform)
}

// RegisterPlayStation5Shader registers a precompiled PlayStation 5 shader for the shader source ID id.
// RegisterPlayStation5Shader panics if a PlayStation 5 shader for id is already registered.
func RegisterPlayStation5Shader(id shaderir.SourceID, vertexHeader, vertexText, pixelHeader, pixelText []byte) {
	theRegistry.registerPlayStation5Shader(id, vertexHeader, vertexText, pixelHeader, pixelText)
}

// PlayStation5Shader returns the precompiled PlayStation 5 shader registered for the shader source ID id.
func PlayStation5Shader(id shaderir.SourceID) (vertexHeader, vertexText, pixelHeader, pixelText []byte, ok bool) {
	return theRegistry.getPlayStation5Shader(id)
}

// RegisterGLSL registers precompiled GLSL and GLSL ES shaders for the shader source ID id.
// Either flavor (vertex/fragment or esVertex/esFragment) may be nil.
// RegisterGLSL panics if GLSL shaders for id are already registered.
func RegisterGLSL(id shaderir.SourceID, vertex, fragment, esVertex, esFragment []byte) {
	theRegistry.registerGLSL(id, vertex, fragment, esVertex, esFragment)
}

// GLSL returns the precompiled GLSL shaders registered for the shader source ID id.
// If es is true, the GLSL ES variant is returned; otherwise the default GLSL variant is returned.
// ok is false when no shader is registered for id, or when the requested variant was registered as nil.
func GLSL(id shaderir.SourceID, es bool) (vertex, fragment []byte, ok bool) {
	return theRegistry.getGLSL(id, es)
}
