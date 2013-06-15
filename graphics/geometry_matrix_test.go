package graphics_test

import (
	"testing"
	. "."
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
	want := AffineMatrixElement(1)
	if want != got {
		t.Errorf("matrix.A() = %f, want %f", got, want)
	}

	got = matrix.B()
	want = AffineMatrixElement(2)
	if want != got {
		t.Errorf("matrix.B() = %f, want %f", got, want)
	}

	got = matrix.C()
	want = AffineMatrixElement(3)
	if want != got {
		t.Errorf("matrix.C() = %f, want %f", got, want)
	}

	got = matrix.D()
	want = AffineMatrixElement(4)
	if want != got {
		t.Errorf("matrix.D() = %f, want %f", got, want)
	}

	got = matrix.Tx()
	want = AffineMatrixElement(5)
	if want != got {
		t.Errorf("matrix.Tx() = %f, want %f", got, want)
	}

	got = matrix.Ty()
	want = AffineMatrixElement(6)
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
