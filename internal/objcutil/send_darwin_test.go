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

package objcutil_test

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/objcutil"
)

func loadFoundation(t *testing.T) {
	t.Helper()
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		t.Fatal(err)
	}
}

// newNSObjects returns two NSObjects that are released when the test finishes.
func newNSObjects(t *testing.T) (obj, other objc.ID) {
	t.Helper()
	loadFoundation(t)
	selNew := objc.RegisterName("new")
	selRelease := objc.RegisterName("release")

	obj = objc.ID(objc.GetClass("NSObject")).Send(selNew)
	other = objc.ID(objc.GetClass("NSObject")).Send(selNew)
	t.Cleanup(func() {
		obj.Send(selRelease)
		other.Send(selRelease)
	})
	return obj, other
}

// newRectValue returns an autoreleased NSValue holding rect and a rectValue key that is released when the test finishes.
// Call it after loadFoundation, inside an autorelease pool on a locked OS thread.
func newRectValue(t *testing.T, rect cocoa.NSRect) (value, key objc.ID) {
	t.Helper()
	selRelease := objc.RegisterName("release")
	selValueWithRect := objc.RegisterName("valueWithRect:")

	value = objc.ID(objc.GetClass("NSValue")).Send(selValueWithRect, rect)
	key = cocoa.NSString_alloc().InitWithUTF8String("rectValue").ID
	t.Cleanup(func() {
		key.Send(selRelease)
	})
	return value, key
}

func TestSend(t *testing.T) {
	obj, other := newNSObjects(t)
	selHash := objc.RegisterName("hash")
	selRetainCount := objc.RegisterName("retainCount")
	selIsEqual := objc.RegisterName("isEqual:")

	if got, want := objcutil.Send(obj, selHash), uintptr(obj.Send(selHash)); got != want {
		t.Errorf("hash: got %d, want %d", got, want)
	}
	if got, want := objcutil.Send(obj, selRetainCount), uintptr(obj.Send(selRetainCount)); got != want {
		t.Errorf("retainCount: got %d, want %d", got, want)
	}
	if got := objcutil.Send(obj, selIsEqual, uintptr(obj)) & 0xff; got != 1 {
		t.Errorf("isEqual: itself: got %d, want 1", got)
	}
	if got := objcutil.Send(obj, selIsEqual, uintptr(other)) & 0xff; got != 0 {
		t.Errorf("isEqual: another object: got %d, want 0", got)
	}
}

func TestValueForKey(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	loadFoundation(t)
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	want := cocoa.NSRect{
		Origin: cocoa.NSPoint{X: 1.5, Y: 2.5},
		Size:   cocoa.NSSize{Width: 3.5, Height: 4.5},
	}
	value, keyRectValue := newRectValue(t, want)

	var got cocoa.NSRect
	if !objcutil.ValueForKey(value, keyRectValue, unsafe.Pointer(&got), unsafe.Sizeof(got)) {
		t.Fatal("rectValue: no value")
	}
	if got != want {
		t.Errorf("rectValue: got %+v, want %+v", got, want)
	}
}
