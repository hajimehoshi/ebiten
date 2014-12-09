package ebiten

import (
	"image/color"
	"math"
)

const ColorMatrixDim = 5

type ColorMatrix struct {
	Elements [ColorMatrixDim - 1][ColorMatrixDim]float64
}

func ColorMatrixI() ColorMatrix {
	return ColorMatrix{
		[ColorMatrixDim - 1][ColorMatrixDim]float64{
			{1, 0, 0, 0, 0},
			{0, 1, 0, 0, 0},
			{0, 0, 1, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

func (c *ColorMatrix) dim() int {
	return ColorMatrixDim
}

func (c *ColorMatrix) Concat(other ColorMatrix) {
	result := ColorMatrix{}
	mul(&other, c, &result)
	*c = result
}

func (c *ColorMatrix) IsIdentity() bool {
	return isIdentity(c)
}

func (c *ColorMatrix) element(i, j int) float64 {
	return c.Elements[i][j]
}

func (c *ColorMatrix) setElement(i, j int, element float64) {
	c.Elements[i][j] = element
}

func Monochrome() ColorMatrix {
	const r float64 = 6968.0 / 32768.0
	const g float64 = 23434.0 / 32768.0
	const b float64 = 2366.0 / 32768.0
	return ColorMatrix{
		[ColorMatrixDim - 1][ColorMatrixDim]float64{
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

func rgba(clr color.Color) (float64, float64, float64, float64) {
	r, g, b, a := clr.RGBA()
	rf := float64(r) / float64(math.MaxUint16)
	gf := float64(g) / float64(math.MaxUint16)
	bf := float64(b) / float64(math.MaxUint16)
	af := float64(a) / float64(math.MaxUint16)
	return rf, gf, bf, af
}

func (c *ColorMatrix) Scale(clr color.Color) {
	rf, gf, bf, af := rgba(clr)
	for i, e := range []float64{rf, gf, bf, af} {
		for j := 0; j < 4; j++ {
			c.Elements[i][j] *= e
		}
	}
}

func (c *ColorMatrix) Translate(clr color.Color) {
	rf, gf, bf, af := rgba(clr)
	c.Elements[0][4] = rf
	c.Elements[1][4] = gf
	c.Elements[2][4] = bf
	c.Elements[3][4] = af
}
