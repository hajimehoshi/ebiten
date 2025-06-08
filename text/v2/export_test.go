// Copyright 2023 The Ebitengine Authors
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

package text

import (
	"golang.org/x/image/math/fixed"
)

func Fixed26_6ToFloat32(x fixed.Int26_6) float32 {
	return fixed26_6ToFloat32(x)
}

func Fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return fixed26_6ToFloat64(x)
}

func Float32ToFixed26_6(x float32) fixed.Int26_6 {
	return float32ToFixed26_6(x)
}

func Float64ToFixed26_6(x float64) fixed.Int26_6 {
	return float64ToFixed26_6(x)
}

type RuneToBoolMap = runeToBoolMap

func (rtb *RuneToBoolMap) Set(rune rune, value bool) {
	rtb.set(rune, value)
}

func (rtb *RuneToBoolMap) Get(rune rune) (bool, bool) {
	return rtb.get(rune)
}
