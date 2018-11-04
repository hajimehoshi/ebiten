// Copyright 2018 The Ebiten Authors
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

// orthoProjectionMatrix returns an orthogonal projection matrix for OpenGL.
//
// The matrix converts the coodinates (left, bottom) - (right, top) to the normalized device coodinates (-1, -1) - (1, 1).
func orthoProjectionMatrix(left, right, bottom, top int) []float32 {
	e11 := 2 / float32(right-left)
	e22 := 2 / float32(top-bottom)
	e14 := -1 * float32(right+left) / float32(right-left)
	e24 := -1 * float32(top+bottom) / float32(top-bottom)

	return []float32{
		e11, 0, 0, 0,
		0, e22, 0, 0,
		0, 0, 1, 0,
		e14, e24, 0, 1,
	}
}
