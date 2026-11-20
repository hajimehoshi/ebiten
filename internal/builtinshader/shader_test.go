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

package builtinshader

import (
	"strings"
	"testing"
)

func TestFilterLinearSamplingAvoidsHalfTexelAdd(t *testing.T) {
	var sources [][]byte
	for address := Address(0); address < AddressCount; address++ {
		sources = append(sources, ShaderSource(FilterLinear, address))
	}
	for _, opaque := range []bool{false, true} {
		sources = append(sources, FinalScreenShaderSource(FilterLinear, opaque))
	}
	for _, src := range sources {
		s := string(src)
		if !strings.Contains(s, "floor(") || strings.Contains(s, "+ 1/2.0") || strings.Contains(s, "fract(") {
			t.Errorf("unexpected linear sampling formula in shader:\n%s", s)
		}
	}
	s := string(ShaderSource(FilterPixelated, AddressUnsafe))
	if !strings.Contains(s, "inversedScale") || !strings.Contains(s, "fract(p1)") {
		t.Errorf("unexpected pixelated sampling formula in shader:\n%s", s)
	}
}
