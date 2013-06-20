package matrix_test

import (
	. "."
	"testing"
)

func TestColorIdentity(t *testing.T) {
	matrix := IdentityColor()
	got := matrix.IsIdentity()
	want := true
	if want != got {
		t.Errorf("matrix.IsIdentity() = %t, want %t", got, want)
	}
}
