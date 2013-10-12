package opengl_test

import (
	. "."
	"testing"
)

func TestAdjustPixels(t *testing.T) {
	pixels := [...]uint8{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	}
	result := AdjustPixels(3, 5, pixels[0:len(pixels)])
	wanted := [...]uint8{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 0, 0, 0,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 0, 0, 0,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 0, 0, 0,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 0, 0, 0,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	if len(wanted) != len(result) {
		t.Errorf("len(result) = %d, wanted %d",
			len(result), len(wanted))
	}
	for i := 0; i < len(result); i++ {
		if wanted[i] != result[i] {
			t.Errorf("result[%d] = %d, wanted %d",
				i, result[i], wanted[i])
		}
	}
}

func TestClp2(t *testing.T) {
	testCases := []struct {
		expected uint64
		arg      uint64
	}{
		{256, 255},
		{256, 256},
		{512, 257},
	}

	for _, testCase := range testCases {
		got := Clp2(testCase.arg)
		wanted := testCase.expected
		if wanted != got {
			t.Errorf("Clp(%d) = %d, wanted %d",
				testCase.arg, got, wanted)
		}

	}
}
