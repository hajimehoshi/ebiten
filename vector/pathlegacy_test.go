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

package vector_test

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestAppendVerticesAndIndicesForFillingVertexLimit(t *testing.T) {
	var path vector.Path
	path.MoveTo(0, 0)
	path.LineTo(1, 0)
	path.LineTo(0, 1)
	path.Close()

	vertices := make([]ebiten.Vertex, 1<<16-4)
	vertices, indices := path.AppendVerticesAndIndicesForFilling(vertices, nil)
	if got, want := len(vertices), 1<<16; got != want {
		t.Errorf("len(vertices) = %d, want %d", got, want)
	}
	if got, want := indices, []uint16{1<<16 - 4, 1<<16 - 3, 1<<16 - 2, 1<<16 - 4, 1<<16 - 2, 1<<16 - 1}; !slices.Equal(got, want) {
		t.Errorf("indices = %v, want %v", got, want)
	}
}

func TestAppendVerticesAndIndicesForFillingTooManyVertices(t *testing.T) {
	var path vector.Path
	path.MoveTo(0, 0)
	path.LineTo(1, 0)
	path.LineTo(0, 1)
	path.Close()

	vertices := make([]ebiten.Vertex, 1<<16-3, 1<<16+1)
	allVertices := vertices[:cap(vertices)]
	allVertices[len(vertices)].SrcX = 1
	indices := make([]uint16, 0, 3)
	allIndices := indices[:cap(indices)]
	allIndices[0] = 1

	func() {
		defer func() {
			if recover() == nil {
				t.Error("AppendVerticesAndIndicesForFilling did not panic")
			}
		}()
		path.AppendVerticesAndIndicesForFilling(vertices, indices)
	}()

	if got, want := allVertices[len(vertices)].SrcX, float32(1); got != want {
		t.Errorf("the vertex slice was modified: got %v, want %v", got, want)
	}
	if got, want := allIndices[0], uint16(1); got != want {
		t.Errorf("the index slice was modified: got %v, want %v", got, want)
	}
}

func TestAppendVerticesAndIndicesForStrokeTooManyVertices(t *testing.T) {
	var path vector.Path
	path.MoveTo(0, 0)
	path.LineTo(1, 0)

	vertices := make([]ebiten.Vertex, 1<<16)
	op := &vector.StrokeOptions{Width: 1}
	defer func() {
		if recover() == nil {
			t.Error("AppendVerticesAndIndicesForStroke did not panic")
		}
	}()
	path.AppendVerticesAndIndicesForStroke(vertices, nil, op)
}
