package matrix

import (
	"math"
)

const GeometryDim = 3

type Geometry struct {
	Elements [GeometryDim - 1][GeometryDim]float64
}

func GeometryI() Geometry {
	return Geometry{
		[GeometryDim - 1][GeometryDim]float64{
			{1, 0, 0},
			{0, 1, 0},
		},
	}
}

func (matrix *Geometry) Dim() int {
	return GeometryDim
}

func (matrix *Geometry) Concat(other Geometry) {
	result := Geometry{}
	mul(&other, matrix, &result)
	*matrix = result
}

func (matrix *Geometry) IsIdentity() bool {
	return isIdentity(matrix)
}

func (matrix *Geometry) element(i, j int) float64 {
	return matrix.Elements[i][j]
}

func (matrix *Geometry) setElement(i, j int, element float64) {
	matrix.Elements[i][j] = element
}

func (matrix *Geometry) Translate(tx, ty float64) {
	matrix.Elements[0][2] += tx
	matrix.Elements[1][2] += ty
}

func (matrix *Geometry) Scale(x, y float64) {
	matrix.Elements[0][0] *= x
	matrix.Elements[0][1] *= x
	matrix.Elements[0][2] *= x
	matrix.Elements[1][0] *= y
	matrix.Elements[1][1] *= y
	matrix.Elements[1][2] *= y
}

func (matrix *Geometry) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	rotate := Geometry{
		[2][3]float64{
			{cos, -sin, 0},
			{sin, cos, 0},
		},
	}
	matrix.Concat(rotate)
}
