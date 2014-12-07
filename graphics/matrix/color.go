package matrix

import (
	"image/color"
	"math"
)

const ColorDim = 5

type Color struct {
	Elements [ColorDim - 1][ColorDim]float64
}

func ColorI() Color {
	return Color{
		[ColorDim - 1][ColorDim]float64{
			{1, 0, 0, 0, 0},
			{0, 1, 0, 0, 0},
			{0, 0, 1, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

func (matrix *Color) dim() int {
	return ColorDim
}

func (matrix *Color) Concat(other Color) {
	result := Color{}
	mul(&other, matrix, &result)
	*matrix = result
}

func (matrix *Color) IsIdentity() bool {
	return isIdentity(matrix)
}

func (matrix *Color) element(i, j int) float64 {
	return matrix.Elements[i][j]
}

func (matrix *Color) setElement(i, j int, element float64) {
	matrix.Elements[i][j] = element
}

func Monochrome() Color {
	const r float64 = 6968.0 / 32768.0
	const g float64 = 23434.0 / 32768.0
	const b float64 = 2366.0 / 32768.0
	return Color{
		[ColorDim - 1][ColorDim]float64{
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

func (matrix *Color) Scale(clr color.Color) {
	rf, gf, bf, af := rgba(clr)
	for i, e := range []float64{rf, gf, bf, af} {
		for j := 0; j < 4; j++ {
			matrix.Elements[i][j] *= e
		}
	}
}

func (matrix *Color) Translate(clr color.Color) {
	rf, gf, bf, af := rgba(clr)
	matrix.Elements[0][4] = rf
	matrix.Elements[1][4] = gf
	matrix.Elements[2][4] = bf
	matrix.Elements[3][4] = af
}
