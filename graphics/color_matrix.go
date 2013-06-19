package graphics

type ColorMatrix struct {
	AffineMatrix
}

const colorMatrixDimension = 5

func NewColorMatrix() *ColorMatrix {
	return &ColorMatrix{*NewAffineMatrix(colorMatrixDimension)}
}

func IdentityColorMatrix() *ColorMatrix {
	return &ColorMatrix{*IdentityAffineMatrix(colorMatrixDimension)}
}

func (matrix *ColorMatrix) Concat(other *ColorMatrix) *ColorMatrix {
	return &ColorMatrix{*matrix.AffineMatrix.Concat(&other.AffineMatrix)}
}

func (matrix *ColorMatrix) Clone() *ColorMatrix {
	return &ColorMatrix{*(matrix.AffineMatrix.Clone())}
}
