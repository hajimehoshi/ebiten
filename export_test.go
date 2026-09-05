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

package ebiten

import "github.com/hajimehoshi/ebiten/v2/internal/builtinshader"

var (
	ImageToBytes = imageToBytes
)

func BuiltinShader(filter builtinshader.Filter, address builtinshader.Address, useColorM bool) *Shader {
	return builtinShader(filter, address, useColorM)
}

func ResetStencilBufferImagesForTesting() {
	stencilBufferM.Lock()
	defer stencilBufferM.Unlock()

	if stencilBufferImage != nil {
		stencilBufferImage.Deallocate()
		stencilBufferImage = nil
	}
	if offscreenImage1 != nil {
		offscreenImage1.Deallocate()
		offscreenImage1 = nil
	}
	if offscreenImage2 != nil {
		offscreenImage2.Deallocate()
		offscreenImage2 = nil
	}
}
