package ebiten_test

import (
	. "."
	"testing"
)

func TestColorIdentity(t *testing.T) {
	ebiten := ColorMatrixI()
	got := ebiten.IsIdentity()
	want := true
	if want != got {
		t.Errorf("matrix.IsIdentity() = %t, want %t", got, want)
	}
}
