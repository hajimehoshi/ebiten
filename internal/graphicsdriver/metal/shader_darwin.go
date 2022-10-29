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

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/msl"
)

type shaderRpsKey struct {
	blend       graphicsdriver.Blend
	stencilMode stencilMode
	screen      bool
}

type Shader struct {
	id graphicsdriver.ShaderID

	ir   *shaderir.Program
	fs   mtl.Function
	vs   mtl.Function
	rpss map[shaderRpsKey]mtl.RenderPipelineState
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
}

func (s *Shader) init(device mtl.Device) error {
	const (
		v = "Vertex"
		f = "Fragment"
	)

	src := msl.Compile(s.ir, v, f)
	lib, err := device.MakeLibrary(src, mtl.CompileOptions{})
	if err != nil {
		return fmt.Errorf("metal: device.MakeLibrary failed: %v, source: %s", err, src)
	}
	vs, err := lib.MakeFunction(v)
	if err != nil {
		return fmt.Errorf("metal: lib.MakeFunction for vertex failed: %v, source: %s", err, src)
	}
	fs, err := lib.MakeFunction(f)
	if err != nil {
		return fmt.Errorf("metal: lib.MakeFunction for fragment failed: %v, source: %s", err, src)
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

	if stencilMode == prepareStencil {
		rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskNone
	} else {
		rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskAll
	}

	rps, err := view.getMTLDevice().MakeRenderPipelineState(rpld)
	if err != nil {
		return mtl.RenderPipelineState{}, err
	}

	s.rpss[key] = rps
	return rps, nil
}
