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

type fxc struct {
	vertex []byte
	pixel  []byte
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

	fxcs                map[shaderir.SourceID]fxc
	metalLibraries      map[shaderir.SourceID]metalLibrary
	playStation5Shaders map[shaderir.SourceID]playStation5Shader
	glslShaders         map[shaderir.SourceID]glslShader
}

var theRegistry registry

func (r *registry) registerFXCs(id shaderir.SourceID, vertex, pixel []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.fxcs == nil {
		r.fxcs = map[shaderir.SourceID]fxc{}
	}
	if _, ok := r.fxcs[id]; ok {
		panic(fmt.Sprintf("shaderprecomp: FXCs for the shader source ID %s are already registered", id))
	}
	r.fxcs[id] = fxc{
		vertex: vertex,
		pixel:  pixel,
	}
}

func (r *registry) getFXCs(id shaderir.SourceID) (vertex, pixel []byte, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, ok := r.fxcs[id]
	return f.vertex, f.pixel, ok
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

// RegisterFXCs registers precompiled FXC binaries for the shader source ID id.
// RegisterFXCs panics if FXCs for id are already registered.
func RegisterFXCs(id shaderir.SourceID, vertex, pixel []byte) {
	theRegistry.registerFXCs(id, vertex, pixel)
}

// FXCs returns the precompiled FXC binaries registered for the shader source ID id.
func FXCs(id shaderir.SourceID) (vertex, pixel []byte, ok bool) {
	return theRegistry.getFXCs(id)
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
