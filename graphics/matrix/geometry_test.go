package matrix_test

import (
	. "."
	"testing"
)

func TestGeometryIdentity(t *testing.T) {
	matrix := IdentityGeometry()
	got := matrix.IsIdentity()
	want := true
	if want != got {
		t.Errorf("matrix.IsIdentity() = %t, want %t", got, want)
	}
}

func TestGeometryConcat(t *testing.T) {
	matrix1 := Geometry{}
	matrix2 := Geometry{}
	matrix1.Elements = [2][3]float64{
		{2, 0, 0},
		{0, 2, 0},
	}
	matrix2.Elements = [2][3]float64{
		{1, 0, 1},
		{0, 1, 1},
	}

	// TODO: 'matrix1x2' may not be a good name.
	matrix1x2 := matrix1
	matrix1x2.Concat(matrix2)
	expected := [][]float64{
		{2, 0, 1},
		{0, 2, 1},
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			got := matrix1x2.Elements[i][j]
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix1x2.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}

	matrix2x1 := matrix2
	matrix2x1.Concat(matrix1)
	expected = [][]float64{
		{2, 0, 2},
		{0, 2, 2},
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			got := matrix2x1.Elements[i][j]
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix2x1.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}
}
