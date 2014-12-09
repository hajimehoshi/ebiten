package ebiten_test

import (
	. "."
	"testing"
)

func TestGeometryIdentity(t *testing.T) {
	ebiten := GeometryMatrixI()
	got := ebiten.IsIdentity()
	want := true
	if want != got {
		t.Errorf("matrix.IsIdentity() = %t, want %t", got, want)
	}
}

func TestGeometryConcat(t *testing.T) {
	matrix1 := GeometryMatrix{}
	matrix2 := GeometryMatrix{}
	matrix1.Elements = [2][3]float64{
		{2, 0, 0},
		{0, 2, 0},
	}
	matrix2.Elements = [2][3]float64{
		{1, 0, 1},
		{0, 1, 1},
	}

	matrix3 := matrix1
	matrix3.Concat(matrix2)
	expected := [][]float64{
		{2, 0, 1},
		{0, 2, 1},
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			got := matrix3.Elements[i][j]
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix3.Element(%d, %d) = %f,"+
					" want %f",
					i, j, got, want)
			}
		}
	}

	matrix4 := matrix2
	matrix4.Concat(matrix1)
	expected = [][]float64{
		{2, 0, 2},
		{0, 2, 2},
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			got := matrix4.Elements[i][j]
			want := expected[i][j]
			if want != got {
				t.Errorf("matrix4.Element(%d, %d) = %f, want %f",
					i, j, got, want)
			}
		}
	}
}
