// Copyright 2022 The Ebiten Authors
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

//go:build ignore
// +build ignore

package main

var Time float
var Cursor vec2
var ScreenSize vec2

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	const (
		blurScaleX  = 0.45
		lowLumScan  = 5.0
		hiLumScan   = 10.0
		BrightBoost = 1.25
		maskDark    = 0.25
		maskFade    = 0.8
	)

	pos := texCoord
	origin, size := imageSrcRegionOnTexture()
	pos -= origin
	pos /= size

	maskFade := 0.3333 * maskFade
	invDims := 1.0 / imageSrcTextureSize().xy
	p := pos * imageSrcTextureSize()
	i := floor(p) + 0.50
	f := p - i
	p = (i + 4.0*f*f*f) * invDims
	p.x = mix(p.x, pos.x, blurScaleX)
	Y := f.y * f.y
	YY := Y * Y
	whichmask := fract(pos.x * -0.4999)
	mask := 1.0
	if whichmask < 0.5 {
		mask -= maskDark
	}

	clr := imageSrc2At(p*size + origin).rgb
	scanLineWeight := (BrightBoost - lowLumScan*(Y-2.05*YY))
	scanLineWeightB := 1.0 - hiLumScan*(YY-2.8*YY*Y)

	return vec4(clr.rgb*mix(scanLineWeight*mask, scanLineWeightB, dot(clr.rgb, vec3(maskFade))), 1.0)
}
