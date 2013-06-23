package matrix

const colorDim = 5

type Color struct {
	Elements [colorDim - 1][colorDim]float64
}

func IdentityColor() Color {
	return Color{
		[colorDim - 1][colorDim]float64{
			{1, 0, 0, 0, 0},
			{0, 1, 0, 0, 0},
			{0, 0, 1, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}

func (matrix *Color) Dim() int {
	return colorDim
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
		[colorDim - 1][colorDim]float64{
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{r, g, b, 0, 0},
			{0, 0, 0, 1, 0},
		},
	}
}
