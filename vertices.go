// Copyright 2017 The Ebiten Authors
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

import (
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

// texelAdjustment represents a number to be used to adjust texel.
// Texels are adjusted by amount propotional to inverse of texelAdjustment.
// This is necessary not to use unexpected pixels outside of texels.
// See #317.
const texelAdjustment = 256

var quadFloat32Num = graphics.QuadVertexSizeInBytes() / 4
