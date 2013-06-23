package matrix

import (
	"math"
)

const geometryDim = 3

type Geometry struct {
	Elements [geometryDim - 1][geometryDim]float64
}

func IdentityGeometry() Geometry {
	return Geometry{
		[geometryDim - 1][geometryDim]float64{
			{1, 0, 0},
			{0, 1, 0},
		},
	}
}

func (matrix *Geometry) Dim() int {
	return geometryDim
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
