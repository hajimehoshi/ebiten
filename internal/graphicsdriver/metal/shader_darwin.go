// Copyright 2020 The Ebiten Authors
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

package metal

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/msl"
)

type precompiledLibraries struct {
	binaries map[shaderir.SourceHash][]byte
	m        sync.Mutex
}

func (c *precompiledLibraries) put(hash shaderir.SourceHash, bin []byte) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.binaries == nil {
		c.binaries = map[shaderir.SourceHash][]byte{}
	}
	if _, ok := c.binaries[hash]; ok {
		panic(fmt.Sprintf("metal: the precompiled library for the hash %s is already registered", hash.String()))
	}
	c.binaries[hash] = bin
}

func (c *precompiledLibraries) get(hash shaderir.SourceHash) []byte {
	c.m.Lock()
	defer c.m.Unlock()

	return c.binaries[hash]
}

var thePrecompiledLibraries precompiledLibraries

func RegisterPrecompiledLibrary(source []byte, bin []byte) {
	thePrecompiledLibraries.put(shaderir.CalcSourceHash(source), bin)
}

type shaderRpsKey struct {
	blend       graphicsdriver.Blend
	stencilMode stencilMode
	screen      bool
}

type Shader struct {
	id graphicsdriver.ShaderID

	ir   *shaderir.Program
	lib  mtl.Library
	fs   mtl.Function
	vs   mtl.Function
	rpss map[shaderRpsKey]mtl.RenderPipelineState

	libraryPrecompiled bool
}

func newShader(device mtl.Device, id graphicsdriver.ShaderID, program *shaderir.Program) (*Shader, error) {
	s := &Shader{
		id:   id,
		ir:   program,
		rpss: map[shaderRpsKey]mtl.RenderPipelineState{},
	}
	if err := s.init(device); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Shader) ID() graphicsdriver.ShaderID {
	return s.id
}

func (s *Shader) Dispose() {
	for _, rps := range s.rpss {
		rps.Release()
	}
	s.vs.Release()
	s.fs.Release()
	// Do not release s.lib if this is precompiled. This is a shared precompiled library.
	if !s.libraryPrecompiled {
		s.lib.Release()
	}
}

func (s *Shader) init(device mtl.Device) error {
	var src string
	if libBin := thePrecompiledLibraries.get(s.ir.SourceHash); len(libBin) > 0 {
		lib, err := device.NewLibraryWithData(libBin)
		if err != nil {
			return err
		}
		s.lib = lib
	} else {
		src = msl.Compile(s.ir)
		lib, err := device.NewLibraryWithSource(src, mtl.CompileOptions{})
		if err != nil {
			return fmt.Errorf("metal: device.MakeLibrary failed: %w, source: %s", err, src)
		}
		s.lib = lib
	}

	vs, err := s.lib.NewFunctionWithName(msl.VertexName)
	if err != nil {
		if src != "" {
			return fmt.Errorf("metal: lib.MakeFunction for vertex failed: %w, source: %s", err, src)
		}
		return fmt.Errorf("metal: lib.MakeFunction for vertex failed: %w", err)
	}
	fs, err := s.lib.NewFunctionWithName(msl.FragmentName)
	if err != nil {
		if src != "" {
			return fmt.Errorf("metal: lib.MakeFunction for fragment failed: %w, source: %s", err, src)
		}
		return fmt.Errorf("metal: lib.MakeFunction for fragment failed: %w", err)
	}
	s.fs = fs
	s.vs = vs
	return nil
}

func (s *Shader) RenderPipelineState(view *view, blend graphicsdriver.Blend, stencilMode stencilMode, screen bool) (mtl.RenderPipelineState, error) {
	key := shaderRpsKey{
		blend:       blend,
		stencilMode: stencilMode,
		screen:      screen,
	}
	if rps, ok := s.rpss[key]; ok {
		return rps, nil
	}

	rpld := mtl.RenderPipelineDescriptor{
		VertexFunction:   s.vs,
		FragmentFunction: s.fs,
	}
	if stencilMode != noStencil {
		rpld.StencilAttachmentPixelFormat = mtl.PixelFormatStencil8
	}

	// TODO: For the precise pixel format, whether the render target is the screen or not must be considered.
	pix := mtl.PixelFormatRGBA8UNorm
	if screen {
		pix = view.colorPixelFormat()
	}
	rpld.ColorAttachments[0].PixelFormat = pix
	rpld.ColorAttachments[0].BlendingEnabled = true

	rpld.ColorAttachments[0].DestinationAlphaBlendFactor = blendFactorToMetalBlendFactor(blend.BlendFactorDestinationAlpha)
	rpld.ColorAttachments[0].DestinationRGBBlendFactor = blendFactorToMetalBlendFactor(blend.BlendFactorDestinationRGB)
	rpld.ColorAttachments[0].SourceAlphaBlendFactor = blendFactorToMetalBlendFactor(blend.BlendFactorSourceAlpha)
	rpld.ColorAttachments[0].SourceRGBBlendFactor = blendFactorToMetalBlendFactor(blend.BlendFactorSourceRGB)
	rpld.ColorAttachments[0].AlphaBlendOperation = blendOperationToMetalBlendOperation(blend.BlendOperationAlpha)
	rpld.ColorAttachments[0].RGBBlendOperation = blendOperationToMetalBlendOperation(blend.BlendOperationRGB)

	if stencilMode == noStencil || stencilMode == drawWithStencil {
		rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskAll
	} else {
		rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskNone
	}

	rps, err := view.getMTLDevice().NewRenderPipelineStateWithDescriptor(rpld)
	if err != nil {
		return mtl.RenderPipelineState{}, err
	}

	s.rpss[key] = rps
	return rps, nil
}
