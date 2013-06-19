package graphics_test

import (
	"testing"
	. "."
)

func setElements(matrix *AffineMatrix, elements [][]float64) {
	dimension := len(elements) + 1
	for i := 0; i < dimension-1; i++ {
		for j := 0; j < dimension; j++ {
			matrix.SetElement(i, j, elements[i][j])
		}
	}
}

func TestAffineMatrixElement(t *testing.T) {
	matrix := NewAffineMatrix(4)
	matrix.SetElement(0, 0, 1)
	matrix.SetElement(0, 1, 2)
	matrix.SetElement(0, 2, 3)
	expected := [][]float64{
		{1, 2, 3, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 1},
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			got := matrix.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}

	matrix.SetElement(1, 0, 4)
	matrix.SetElement(1, 1, 5)
	matrix.SetElement(2, 3, 6)
	expected = [][]float64{
		{1, 2, 3, 0},
		{4, 5, 0, 0},
		{0, 0, 0, 6},
		{0, 0, 0, 1},
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			got := matrix.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}
}

func TestAffineMatrixIsIdentity(t *testing.T) {
	matrix := NewAffineMatrix(4)
	matrix.SetElement(0, 0, 1)
	matrix.SetElement(1, 1, 1)
	matrix.SetElement(2, 2, 1)
	got := matrix.IsIdentity()
	want := true
	if want != got {
		t.Errorf("matrix.IsIdentity() = %t, want %t", got, want)
	}

	matrix2 := matrix.Clone()
	got = matrix2.IsIdentity()
	want = true
	if want != got {
		t.Errorf("matrix2.IsIdentity() = %t, want %t", got, want)
	}

	matrix2.SetElement(0, 1, 1)
	got = matrix2.IsIdentity()
	want = false
	if want != got {
		t.Errorf("matrix2.IsIdentity() = %t, want %t", got, want)
	}
}

func TestAffineMatrixConcat(t *testing.T) {
	matrix1 := IdentityAffineMatrix(3)
	matrix2 := IdentityAffineMatrix(3)
	setElements(matrix1, [][]float64{
		{2, 0, 0},
		{0, 2, 0},
	})
	setElements(matrix2, [][]float64{
		{1, 0, 1},
		{0, 1, 1},
	})

	// TODO: 'matrix1x2' may not be a good name.
	matrix1x2 := matrix1.Concat(matrix2)
	expected := [][]float64{
		{2, 0, 1},
		{0, 2, 1},
		{0, 0, 1},
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			got := matrix1x2.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix1x2.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}

	matrix2x1 := matrix2.Concat(matrix1)
	expected = [][]float64{
		{2, 0, 2},
		{0, 2, 2},
		{0, 0, 1},
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			got := matrix2x1.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix2x1.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}

	matrix3 := NewAffineMatrix(4)
	matrix4 := NewAffineMatrix(4)
	setElements(matrix3, [][]float64{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
	})
	setElements(matrix4, [][]float64{
		{13, 14, 15, 16},
		{17, 18, 19, 20},
		{21, 22, 23, 24},
	})

	matrix3x4 := matrix3.Concat(matrix4)
	expected = [][]float64{
		{218, 260, 302, 360},
		{278, 332, 386, 460},
		{338, 404, 470, 560},
		{0, 0, 0, 1}}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			got := matrix3x4.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix3x4.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}

	matrix4x3 := matrix4.Concat(matrix3)
	expected = [][]float64{
		{110, 116, 122, 132},
		{314, 332, 350, 376},
		{518, 548, 578, 620},
		{0, 0, 0, 1}}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			got := matrix4x3.Element(i, j)
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix4x3.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}
}
