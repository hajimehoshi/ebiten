package graphics

type AffineMatrixElement float64

type AffineMatrix struct {
	elements  []AffineMatrixElement
	dimension int
}

func NewAffineMatrix(dimension int) *AffineMatrix {
	if dimension < 0 {
		panic("invalid dimension")
	}
	matrix := &AffineMatrix{}
	elementsNumber := dimension * (dimension - 1)
	matrix.elements = make([]AffineMatrixElement, elementsNumber)
	matrix.dimension = dimension
	return matrix
}

func IdentityAffineMatrix(dimension int) *AffineMatrix {
	if dimension < 0 {
		panic("invalid dimension")
	}
	matrix := NewAffineMatrix(dimension)
	for i := 0; i < dimension-1; i++ {
		for j := 0; j < dimension; j++ {
			if i == j {
				matrix.elements[i*dimension+j] = 1
			} else {
				matrix.elements[i*dimension+j] = 0
			}
		}
	}
	return matrix
}

func (matrix *AffineMatrix) Clone() *AffineMatrix {
	result := NewAffineMatrix(matrix.dimension)
	copy(result.elements, matrix.elements)
	return result
}

func (matrix *AffineMatrix) Element(i, j int) AffineMatrixElement {
	dimension := matrix.dimension
	if i < 0 || dimension <= i {
		panic("out of range index i")
	}
	if j < 0 || dimension <= j {
		panic("out of range index j")
	}
	if i == dimension-1 {
		if j == dimension-1 {
			return 1
		}
		return 0
	}
	return matrix.elements[i*dimension+j]
}

func (matrix *AffineMatrix) SetElement(i, j int, element AffineMatrixElement) {
	dimension := matrix.dimension
	if i < 0 || dimension-1 <= i {
		panic("out of range index i")
	}
	if j < 0 || dimension <= j {
		panic("out of range index j")
	}
	matrix.elements[i*dimension+j] = element
}

func (matrix *AffineMatrix) IsIdentity() bool {
	dimension := matrix.dimension
	for i := 0; i < dimension-1; i++ {
		for j := 0; j < dimension; j++ {
			element := matrix.elements[i*dimension+j]
			if i == j && element != 1 {
				return false
			} else if i != j && element != 0 {
				return false
			}
		}
	}
	return true
}

/*
 * TODO: The arguments' names are strange even though they are not wrong.
 */
func (rhs *AffineMatrix) Concat(lhs *AffineMatrix) *AffineMatrix {
	dimension := lhs.dimension
	if dimension != rhs.dimension {
		panic("diffrent-sized matrices can't be concatenated")
	}
	result := NewAffineMatrix(dimension)

	for i := 0; i < dimension-1; i++ {
		for j := 0; j < dimension; j++ {
			var element AffineMatrixElement = 0.0
			for k := 0; k < dimension-1; k++ {
				element += lhs.elements[i*dimension+k] *
					rhs.elements[k*dimension+j]
			}
			if j == dimension-1 {
				element += lhs.elements[i*dimension+dimension-1]
			}
			result.elements[i*dimension+j] = element
		}
	}

	return result
}
