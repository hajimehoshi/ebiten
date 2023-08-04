// Copyright 2022 The Ebitengine Authors
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
	"fmt"
	"image/color"
)

// ColorScale represents a scale of RGBA color.
// ColorScale is intended to be applied to a premultiplied-alpha color value.
//
// The initial (zero) value of ColorScale is an identity scale (1, 1, 1, 1).
type ColorScale struct {
	// These values are adjusted by -1 from the actual values.
	// It's because the initial value should be 1 instead of 0.
	r_1, g_1, b_1, a_1 float32
}

// String returns a string representing the color scale.
func (c *ColorScale) String() string {
	return fmt.Sprintf("(%f,%f,%f,%f)", c.r_1+1, c.g_1+1, c.b_1+1, c.a_1+1)
}

// Reset resets the ColorScale as identity.
func (c *ColorScale) Reset() {
	c.r_1 = 0
	c.g_1 = 0
	c.b_1 = 0
	c.a_1 = 0
}

// R returns the red scale.
func (c *ColorScale) R() float32 {
	return c.r_1 + 1
}

// G returns the green scale.
func (c *ColorScale) G() float32 {
	return c.g_1 + 1
}

// B returns the blue scale.
func (c *ColorScale) B() float32 {
	return c.b_1 + 1
}

// A returns the alpha scale.
func (c *ColorScale) A() float32 {
	return c.a_1 + 1
}

func (c *ColorScale) elements() (float32, float32, float32, float32) {
	return c.r_1 + 1, c.g_1 + 1, c.b_1 + 1, c.a_1 + 1
}

// SetR overwrites the current red value with r.
func (c *ColorScale) SetR(r float32) {
	c.r_1 = r - 1
}

// SetG overwrites the current green value with g.
func (c *ColorScale) SetG(g float32) {
	c.g_1 = g - 1
}

// SetB overwrites the current blue value with b.
func (c *ColorScale) SetB(b float32) {
	c.b_1 = b - 1
}

// SetA overwrites the current alpha value with a.
func (c *ColorScale) SetA(a float32) {
	c.a_1 = a - 1
}

// Scale multiplies the given values to the current scale.
//
// Scale is slightly different from colorm.ColorM's Scale in terms of alphas.
// ColorScale is applied to premultiplied-alpha colors, while colorm.ColorM is applied to straight-alpha colors.
// Thus, ColorM.Scale(r, g, b, a) equals to ColorScale.Scale(r*a, g*a, b*a, a).
func (c *ColorScale) Scale(r, g, b, a float32) {
	c.r_1 = (c.r_1+1)*r - 1
	c.g_1 = (c.g_1+1)*g - 1
	c.b_1 = (c.b_1+1)*b - 1
	c.a_1 = (c.a_1+1)*a - 1
}

// ScaleAlpha multiplies the given alpha value to the current scale.
func (c *ColorScale) ScaleAlpha(a float32) {
	c.r_1 = (c.r_1+1)*a - 1
	c.g_1 = (c.g_1+1)*a - 1
	c.b_1 = (c.b_1+1)*a - 1
	c.a_1 = (c.a_1+1)*a - 1
}

// ScaleWithColor multiplies the given color values to the current scale.
func (c *ColorScale) ScaleWithColor(clr color.Color) {
	cr, cg, cb, ca := clr.RGBA()
	c.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)
}

// ScaleWithColorScale multiplies the given color scale values to the current scale.
func (c *ColorScale) ScaleWithColorScale(colorScale ColorScale) {
	c.r_1 = (c.r_1+1)*(colorScale.r_1+1) - 1
	c.g_1 = (c.g_1+1)*(colorScale.g_1+1) - 1
	c.b_1 = (c.b_1+1)*(colorScale.b_1+1) - 1
	c.a_1 = (c.a_1+1)*(colorScale.a_1+1) - 1
}

func (c *ColorScale) apply(r, g, b, a float32) (float32, float32, float32, float32) {
	return (c.r_1 + 1) * r, (c.g_1 + 1) * g, (c.b_1 + 1) * b, (c.a_1 + 1) * a
}
