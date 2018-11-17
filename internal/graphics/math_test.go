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

package graphics_test

import (
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/graphics"
)

func TestNextPowerOf2(t *testing.T) {
	testCases := []struct {
		expected int
		arg      int
	}{
		{256, 255},
		{256, 256},
		{512, 257},
	}

	for _, testCase := range testCases {
		got := NextPowerOf2Int(testCase.arg)
		wanted := testCase.expected
		if wanted != got {
			t.Errorf("Clp(%d) = %d, wanted %d", testCase.arg, got, wanted)
		}

	}
}
