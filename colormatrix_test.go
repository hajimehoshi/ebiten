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

package ebiten_test

import (
	. "."
	"testing"
)

func TestColorInit(t *testing.T) {
	var m ColorMatrix
	for i := 0; i < ColorMatrixDim-1; i++ {
		for j := 0; j < ColorMatrixDim; j++ {
			got := m.Element(i, j)
			want := 0.0
			if i == j {
				want = 1
			}
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}

	m.Add(m)
	for i := 0; i < ColorMatrixDim-1; i++ {
		for j := 0; j < ColorMatrixDim; j++ {
			got := m.Element(i, j)
			want := 0.0
			if i == j {
				want = 2
			}
			if want != got {
				t.Errorf("m.Element(%d, %d) = %f, want %f", i, j, got, want)
			}
		}
	}
}
