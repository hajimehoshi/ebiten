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

import (
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics"
)

var (
	theVerticesBackend = &verticesBackend{}
)

type verticesBackend struct {
	backend []float32
	head    int
	m       sync.Mutex
}

func (v *verticesBackend) slice(n int) []float32 {
	v.m.Lock()

	need := n * graphics.VertexFloatNum
	if v.head+need > len(v.backend) {
		v.backend = nil
		v.head = 0
	}

	if v.backend == nil {
		l := 1024
		if n > l {
			l = n
		}
		v.backend = make([]float32, graphics.VertexFloatNum*l)
	}

	s := v.backend[v.head : v.head+need]
	v.head += need

	v.m.Unlock()
	return s
}

func vertexSlice(n int) []float32 {
	return theVerticesBackend.slice(n)
}
