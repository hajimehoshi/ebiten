package ebiten

import (
	"math"
)

const GeometryDim = 3

type GeometryMatrix struct {
	Elements [GeometryDim - 1][GeometryDim]float64
}

func GeometryMatrixI() GeometryMatrix {
	return GeometryMatrix{
		[GeometryDim - 1][GeometryDim]float64{
			{1, 0, 0},
			{0, 1, 0},
		},
	}
}

func (g *GeometryMatrix) dim() int {
	return GeometryDim
}

func (g *GeometryMatrix) Concat(other GeometryMatrix) {
	result := GeometryMatrix{}
	mul(&other, g, &result)
	*g = result
}

func (g *GeometryMatrix) IsIdentity() bool {
	return isIdentity(g)
}

func (g *GeometryMatrix) element(i, j int) float64 {
	return g.Elements[i][j]
}

func (g *GeometryMatrix) setElement(i, j int, element float64) {
	g.Elements[i][j] = element
}

func (g *GeometryMatrix) Translate(tx, ty float64) {
	g.Elements[0][2] += tx
	g.Elements[1][2] += ty
}

func (g *GeometryMatrix) Scale(x, y float64) {
	g.Elements[0][0] *= x
	g.Elements[0][1] *= x
	g.Elements[0][2] *= x
	g.Elements[1][0] *= y
	g.Elements[1][1] *= y
	g.Elements[1][2] *= y
}

func (g *GeometryMatrix) Rotate(theta float64) {
	sin, cos := math.Sincos(theta)
	rotate := GeometryMatrix{
		[2][3]float64{
			{cos, -sin, 0},
			{sin, cos, 0},
		},
	}
	g.Concat(rotate)
}
