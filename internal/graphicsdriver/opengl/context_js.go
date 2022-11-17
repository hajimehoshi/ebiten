// Copyright 2014 Hajime Hoshi
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

package opengl

import (
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

type contextImpl struct {
	canvas js.Value
	webGL2 bool
}

func (*context) stencilFormat() uint32 {
	// gl.DEPTH24_STENCIL8 is not available on WebGL1.
	return gl.STENCIL_INDEX8
}
