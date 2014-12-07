package shader_test

import (
	. "."
	"testing"
)

func TestNextPowerOf2(t *testing.T) {
	testCases := []struct {
		expected uint64
		arg      uint64
	}{
		{256, 255},
		{256, 256},
		{512, 257},
	}

	for _, testCase := range testCases {
		got := NextPowerOf2(testCase.arg)
		wanted := testCase.expected
		if wanted != got {
			t.Errorf("Clp(%d) = %d, wanted %d",
				testCase.arg, got, wanted)
		}

	}
}
