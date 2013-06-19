package graphics

import (
	"math"
)

type GeometryMatrix struct {
	AffineMatrix
}

const geometryMatrixDimension = 3

func NewGeometryMatrix() *GeometryMatrix {
	return &GeometryMatrix{*NewAffineMatrix(geometryMatrixDimension)}
}

func IdentityGeometryMatrix() *GeometryMatrix {
	return &GeometryMatrix{*IdentityAffineMatrix(geometryMatrixDimension)}
}

func TranslateMatrix(tx, ty float64) *GeometryMatrix {
	matrix := IdentityGeometryMatrix()
	matrix.SetTx(tx)
	matrix.SetTy(ty)
	return matrix
}

func RotateMatrix(theta float64) *GeometryMatrix {
	matrix := NewGeometryMatrix()
	cos, sin := math.Cos(theta), math.Sin(theta)
	matrix.SetA(cos)
	matrix.SetB(-sin)
	matrix.SetC(sin)
	matrix.SetD(cos)
	return matrix
}

func (matrix *GeometryMatrix) Concat(other *GeometryMatrix) *GeometryMatrix {
	return &GeometryMatrix{*matrix.AffineMatrix.Concat(&other.AffineMatrix)}
}

func (matrix *GeometryMatrix) Clone() *GeometryMatrix {
	return &GeometryMatrix{*(matrix.AffineMatrix.Clone())}
}

func (matrix *GeometryMatrix) A() float64 {
	return matrix.Element(0, 0)
}

func (matrix *GeometryMatrix) B() float64 {
	return matrix.Element(0, 1)
}

func (matrix *GeometryMatrix) C() float64 {
	return matrix.Element(1, 0)
}

func (matrix *GeometryMatrix) D() float64 {
	return matrix.Element(1, 1)
}

func (matrix *GeometryMatrix) Tx() float64 {
	return matrix.Element(0, 2)
}

func (matrix *GeometryMatrix) Ty() float64 {
	return matrix.Element(1, 2)
}

func (matrix *GeometryMatrix) SetA(a float64) {
	matrix.SetElement(0, 0, a)
}

func (matrix *GeometryMatrix) SetB(b float64) {
	matrix.SetElement(0, 1, b)
}

func (matrix *GeometryMatrix) SetC(c float64) {
	matrix.SetElement(1, 0, c)
}

func (matrix *GeometryMatrix) SetD(d float64) {
	matrix.SetElement(1, 1, d)
}

func (matrix *GeometryMatrix) SetTx(tx float64) {
	matrix.SetElement(0, 2, tx)
}

func (matrix *GeometryMatrix) SetTy(ty float64) {
	matrix.SetElement(1, 2, ty)
}
