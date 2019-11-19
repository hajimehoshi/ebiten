// Copyright 2019 The Ebiten Authors
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

// +build js android ios

package opengl

const canUsePBO = false

type pboState struct{}

var thePBOState pboState

func (s *pboState) mapPBO(img *Image) {
	panic("opengl: PBO is not available in this environment")
}

func (s *pboState) draw(pix []byte, x, y, width, height int) {
	panic("opengl: PBO is not available in this environment")
}

func (s *pboState) unmapPBO() {
	panic("opengl: PBO is not available in this environment")
}
