// Copyright 2023 The Ebitengine Authors
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

package directx

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type pipelineStateKey struct {
	blend       graphicsdriver.Blend
	stencilMode stencilMode
	screen      bool
}

type shader12 struct {
	graphics       *graphics12
	id             graphicsdriver.ShaderID
	uniformTypes   []shaderir.Type
	uniformOffsets []int
	vertexShader   *_ID3DBlob
	pixelShader    *_ID3DBlob

	pipelineStates map[pipelineStateKey]*_ID3D12PipelineState
}

func (s *shader12) ID() graphicsdriver.ShaderID {
	return s.id
}

func (s *shader12) Dispose() {
	s.graphics.removeShader(s)
}

func (s *shader12) disposeImpl() {
	for c, p := range s.pipelineStates {
		p.Release()
		delete(s.pipelineStates, c)
	}

	if s.pixelShader != nil {
		s.pixelShader.Release()
		s.pixelShader = nil
	}
	if s.vertexShader != nil {
		count := s.vertexShader.Release()
		if count == 0 {
			for k, v := range vertexShaderCache {
				if v == s.vertexShader {
					delete(vertexShaderCache, k)
				}
			}
		}
		s.vertexShader = nil
	}
}

func (s *shader12) pipelineState(blend graphicsdriver.Blend, stencilMode stencilMode, screen bool) (*_ID3D12PipelineState, error) {
	key := pipelineStateKey{
		blend:       blend,
		stencilMode: stencilMode,
		screen:      screen,
	}
	if state, ok := s.pipelineStates[key]; ok {
		return state, nil
	}

	state, err := s.graphics.pipelineStates.newPipelineState(s.graphics.device, s.vertexShader, s.pixelShader, blend, stencilMode, screen)
	if err != nil {
		return nil, err
	}
	if s.pipelineStates == nil {
		s.pipelineStates = map[pipelineStateKey]*_ID3D12PipelineState{}
	}
	s.pipelineStates[key] = state
	return state, nil
}
