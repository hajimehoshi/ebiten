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

//go:build !race

// The allocation tests are excluded from race builds: with -race, sync.Pool.Put deliberately drops entries, so these
// calls can allocate.

package objcutil_test

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/objcutil"
)

func TestSendAllocs(t *testing.T) {
	obj, other := newNSObjects(t)
	selHash := objc.RegisterName("hash")
	selIsEqual := objc.RegisterName("isEqual:")

	if n := testing.AllocsPerRun(100, func() {
		objcutil.Send(obj, selHash)
		objcutil.Send(obj, selIsEqual, uintptr(other))
	}); n != 0 {
		t.Errorf("allocations: got %v, want 0", n)
	}
}

func TestValueForKeyAllocs(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	loadFoundation(t)
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	value, keyRectValue := newRectValue(t, cocoa.NSRect{
		Origin: cocoa.NSPoint{X: 1.5, Y: 2.5},
		Size:   cocoa.NSSize{Width: 3.5, Height: 4.5},
	})

	if n := testing.AllocsPerRun(100, func() {
		var got cocoa.NSRect
		objcutil.ValueForKey(value, keyRectValue, unsafe.Pointer(&got), unsafe.Sizeof(got))
	}); n != 0 {
		t.Errorf("allocations: got %v, want 0", n)
	}
}
