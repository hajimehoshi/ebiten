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

//go:build darwin && !ios

package metal

import (
	"bytes"
	"image"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

func TestReadPixelsAsync(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if systemDefaultDeviceErr != nil {
		t.Fatal(systemDefaultDeviceErr)
	}
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	cq, err := systemDefaultDevice.NewCommandQueue()
	if err != nil {
		t.Fatal(err)
	}
	defer cq.Release()
	g := &Graphics{cq: cq, view: view{device: systemDefaultDevice}}
	defer func() {
		g.flushCommandBufferIfNeeded(false)
		for _, cbs := range g.frameToCB {
			for _, cb := range cbs {
				cb.WaitUntilCompleted()
				cb.Release()
			}
		}
	}()
	for _, discard := range []bool{false, true} {
		img, err := g.NewImage(17, 11)
		if err != nil {
			t.Fatal(err)
		}
		i := img.(*Image)
		region := image.Rect(0, 0, 17, 11)
		original := bytes.Repeat([]byte{123, 45, 67, 255}, region.Dx()*region.Dy())
		if err := i.WritePixels([]graphicsdriver.PixelsArgs{{Pixels: original, Region: region}}); err != nil {
			t.Fatal(err)
		}
		args := []graphicsdriver.PixelsArgs{
			{
				Pixels: make([]byte, 3*5*4),
				Region: image.Rect(2, 3, 5, 8),
			},
			{
				Pixels: make([]byte, 17*11*4),
				Region: region,
			},
		}
		if _, err := i.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{Pixels: make([]byte, 1), Region: region}}); err == nil {
			t.Fatal("invalid length accepted")
		}
		result, err := i.ReadPixelsAsync(args)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := result.(*readback); !ok {
			t.Fatal("not a native Metal read-back")
		}
		if err := i.WritePixels([]graphicsdriver.PixelsArgs{{Pixels: make([]byte, len(original)), Region: region}}); err != nil {
			t.Fatal(err)
		}
		i.Dispose()
		if discard {
			result.Dispose()
			continue
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			done, err := result.Poll()
			if err != nil {
				result.Dispose()
				t.Fatal(err)
			}
			if done {
				break
			}
			if time.Now().After(deadline) {
				result.Dispose()
				t.Fatal("read-back did not complete without a frame flush")
			}
			time.Sleep(time.Millisecond)
		}
		result.Dispose()
		for _, arg := range args {
			if !bytes.Equal(arg.Pixels, bytes.Repeat([]byte{123, 45, 67, 255}, arg.Region.Dx()*arg.Region.Dy())) {
				t.Errorf("incorrect capture for %v", arg.Region)
			}
		}
	}
	r := &readback{commandBuffer: mtl.CommandBuffer{}}
	if done, err := r.Poll(); !done || err != nil {
		t.Errorf("empty read-back: %v, %v", done, err)
	}
	r.Dispose()
}
