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

package svgrender

//ebitengine:shadersource
const gradientShader = `//kage:unit pixels
package main

var Stops [64]vec4
var Offsets [64]float
var Count float
var Inverse [6]float
var Linear vec3
var PaintOrigin vec2
var Center, Focus vec2
var Radius, Radial, LinearRGB, Spread float

func srgb(v float) float {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*pow(v, 1.0/2.4) - 0.055
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	q := srcPos + PaintOrigin
	p := vec2(Inverse[0]*q.x+Inverse[2]*q.y+Inverse[4], Inverse[1]*q.x+Inverse[3]*q.y+Inverse[5])
	var t float
	var outside bool
	if Radial == 0 {
		t = dot(q, Linear.xy) + Linear.z
	} else {
		d := p - Focus
		f := Focus - Center
		a := Radius*Radius - dot(f, f)
		b := dot(d, f)
		c := dot(d, d)
		if a > 0.000001*Radius*Radius {
			disc := sqrt(max(0, b*b+a*c))
			if b < 0 {
				t = c / (disc - b)
			} else {
				t = (b + disc) / a
			}
		} else {
			// Limit of the focal-circle equation when the focus is on its circumference.
			if c == 0 {
				t = 0
			} else if b < 0 {
				t = -c / (2 * b)
			} else {
				t = 1
				outside = true
			}
		}
	}
	if !outside {
		if Spread == 1 {
			t = fract(t)
		} else if Spread == 2 {
			t = 1 - abs(mod(t, 2)-1)
		}
	}
	value := Stops[0]
	for i := 1; i < 64; i++ {
		if float(i) >= Count {
			break
		}
		if t < Offsets[i] {
			if Offsets[i] > Offsets[i-1] {
				value = mix(Stops[i-1], Stops[i], clamp((t-Offsets[i-1])/(Offsets[i]-Offsets[i-1]), 0, 1))
			}
			break
		}
		value = Stops[i]
	}
	if LinearRGB != 0 {
		value.r = srgb(value.r)
		value.g = srgb(value.g)
		value.b = srgb(value.b)
	}
	return vec4(value.rgb*value.a, value.a)
}
`

//ebitengine:shadersource
const resolveShader = `//kage:unit pixels
package main

var Samples float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	origin := imageSrc0Origin()
	base := floor(srcPos-origin)*Samples + origin
	var sum vec4
	for y := range 8 {
		if float(y) >= Samples {
			break
		}
		for x := range 8 {
			if float(x) >= Samples {
				break
			}
			sum += imageSrc0At(base + vec2(float(x)+0.5, float(y)+0.5))
		}
	}
	return sum / (Samples * Samples)
}
`
