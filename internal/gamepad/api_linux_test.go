// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !android && !nintendosdk && !playstation5

package gamepad

import (
	"math"
	"testing"
	"unsafe"
)

// TestFFEffectLayout verifies that ff_effect matches the layout of the
// kernel's struct ff_effect. The struct's size is encoded in the EVIOCSFF
// ioctl request number, so a layout mismatch makes the ioctl fail with
// EINVAL.
func TestFFEffectLayout(t *testing.T) {
	// The parameter union's largest member contains a pointer, so the total
	// size depends on the architecture's pointer size.
	wantSize := uintptr(44)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		wantSize = 48
	}
	if got := unsafe.Sizeof(ff_effect{}); got != wantSize {
		t.Errorf("unsafe.Sizeof(ff_effect{}): got: %d, want: %d", got, wantSize)
	}
	if got, want := unsafe.Offsetof(ff_effect{}.u), uintptr(16); got != want {
		t.Errorf("unsafe.Offsetof(ff_effect{}.u): got: %d, want: %d", got, want)
	}
}

func TestFFMagnitude(t *testing.T) {
	tests := []struct {
		in   float64
		want uint16
	}{
		{math.NaN(), 0},
		{math.Inf(-1), 0},
		{-1, 0},
		{0, 0},
		{0.5, 0x7fff},
		{1, 0xffff},
		{2, 0xffff},
		{math.Inf(1), 0xffff},
	}
	for _, test := range tests {
		if got := ffMagnitude(test.in); got != test.want {
			t.Errorf("ffMagnitude(%v): got: %#x, want: %#x", test.in, got, test.want)
		}
	}
}
