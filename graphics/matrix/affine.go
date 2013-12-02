package matrix

type affine interface {
	Dim() int
	element(i, j int) float64
	setElement(i, j int, element float64)
}

func isIdentity(matrix affine) bool {
	dim := matrix.Dim()
	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			element := matrix.element(i, j)
			if i == j && element != 1 {
				return false
			} else if i != j && element != 0 {
				return false
			}
		}
	}
	return true
}

func mul(lhs, rhs, result affine) {
	dim := lhs.Dim()
	if dim != rhs.Dim() {
		panic("diffrent-sized matrices can't be multiplied")
	}

	for i := 0; i < dim-1; i++ {
		for j := 0; j < dim; j++ {
			element := float64(0)
			for k := 0; k < dim-1; k++ {
				element += lhs.element(i, k) *
					rhs.element(k, j)
			}
			if j == dim-1 {
				element += lhs.element(i, j)
			}
			result.setElement(i, j, element)
		}
	}
}
