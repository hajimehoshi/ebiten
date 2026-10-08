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

package gamepad_test

import (
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

// TestFFEffectLayout verifies that ff_effect matches the layout of the kernel's
// struct ff_effect. The size is encoded in the EVIOCSFF ioctl request number,
// so a layout mismatch makes every effect upload fail.
func TestFFEffectLayout(t *testing.T) {
	// The kernel's largest union member holds a user-space pointer, so the
	// total size depends on the architecture's pointer size.
	wantSize := uintptr(44)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		wantSize = 48
	}
	if got, want := gamepad.FFEffectSize, wantSize; got != want {
		t.Errorf("gamepad.FFEffectSize: got: %d, want: %d", got, want)
	}
	if got, want := gamepad.FFEffectUnionOffset, uintptr(16); got != want {
		t.Errorf("gamepad.FFEffectUnionOffset: got: %d, want: %d", got, want)
	}
	if got, want := gamepad.FFEffectRumbleOffset, uintptr(16); got != want {
		t.Errorf("gamepad.FFEffectRumbleOffset: got: %d, want: %d", got, want)
	}
}
