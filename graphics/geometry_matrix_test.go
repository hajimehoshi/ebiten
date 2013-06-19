package graphics_test

import (
	. "."
	"testing"
)

func TestGeometryMatrixElements(t *testing.T) {
	matrix := NewGeometryMatrix()
	matrix.SetA(1)
	matrix.SetB(2)
	matrix.SetC(3)
	matrix.SetD(4)
	matrix.SetTx(5)
	matrix.SetTy(6)

	got := matrix.A()
	want := float64(1)
	if want != got {
		t.Errorf("matrix.A() = %f, want %f", got, want)
	}

	got = matrix.B()
	want = float64(2)
	if want != got {
		t.Errorf("matrix.B() = %f, want %f", got, want)
	}

	got = matrix.C()
	want = float64(3)
	if want != got {
		t.Errorf("matrix.C() = %f, want %f", got, want)
	}

	got = matrix.D()
	want = float64(4)
	if want != got {
		t.Errorf("matrix.D() = %f, want %f", got, want)
	}

	got = matrix.Tx()
	want = float64(5)
	if want != got {
		t.Errorf("matrix.Tx() = %f, want %f", got, want)
	}

	got = matrix.Ty()
	want = float64(6)
	if want != got {
		t.Errorf("matrix.Ty() = %f, want %f", got, want)
	}
}

func TestGeometryIdentity(t *testing.T) {
	matrix := IdentityGeometryMatrix()
	got := matrix.IsIdentity()
	want := true
	if want != got {
		t.Errorf("matrix.IsIdentity() = %t, want %t", got, want)
	}
}
