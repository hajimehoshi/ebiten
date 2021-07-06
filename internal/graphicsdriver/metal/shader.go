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

//go:build darwin
// +build darwin

package metal

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/metal"
)

type shaderRpsKey struct {
	compositeMode driver.CompositeMode
	stencilMode   stencilMode
}

type Shader struct {
	id driver.ShaderID

	ir   *shaderir.Program
	fs   mtl.Function
	vs   mtl.Function
	rpss map[shaderRpsKey]mtl.RenderPipelineState
}

func newShader(device mtl.Device, id driver.ShaderID, program *shaderir.Program) (*Shader, error) {
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

func (s *Shader) ID() driver.ShaderID {
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

	src := metal.Compile(s.ir, v, f)
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

func (s *Shader) RenderPipelineState(device mtl.Device, compositeMode driver.CompositeMode, stencilMode stencilMode) (mtl.RenderPipelineState, error) {
	if rps, ok := s.rpss[shaderRpsKey{
		compositeMode: compositeMode,
		stencilMode:   stencilMode,
	}]; ok {
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
	rpld.ColorAttachments[0].PixelFormat = mtl.PixelFormatRGBA8UNorm
	rpld.ColorAttachments[0].BlendingEnabled = true

	src, dst := compositeMode.Operations()
	rpld.ColorAttachments[0].DestinationAlphaBlendFactor = operationToBlendFactor(dst)
	rpld.ColorAttachments[0].DestinationRGBBlendFactor = operationToBlendFactor(dst)
	rpld.ColorAttachments[0].SourceAlphaBlendFactor = operationToBlendFactor(src)
	rpld.ColorAttachments[0].SourceRGBBlendFactor = operationToBlendFactor(src)
	if stencilMode == prepareStencil {
		rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskNone
	} else {
		rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskAll
	}

	rps, err := device.MakeRenderPipelineState(rpld)
	if err != nil {
		return mtl.RenderPipelineState{}, err
	}

	s.rpss[shaderRpsKey{
		compositeMode: compositeMode,
		stencilMode:   stencilMode,
	}] = rps
	return rps, nil
}
