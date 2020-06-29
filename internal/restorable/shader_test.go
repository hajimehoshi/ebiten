// Copyright 2018 The Ebiten Authors
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

package restorable_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	. "github.com/hajimehoshi/ebiten/internal/restorable"
	etesting "github.com/hajimehoshi/ebiten/internal/testing"
)

func TestShader(t *testing.T) {
	if !graphicscommand.IsShaderAvailable() {
		t.Skip("shader is not available on this environment")
	}

	img := NewImage(1, 1, false)
	defer img.Dispose()

	ir := etesting.ShaderProgramFill(0xff, 0, 0, 0xff)
	s := NewShader(&ir)
	us := []interface{}{
		[]float32{0, 0},
	}
	img.DrawTriangles(nil, quadVertices(1, 1, 0, 0), graphics.QuadIndices(), nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, s, us)

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := RestoreIfNeeded(); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{0xff, 0, 0, 0xff}
	got := pixelsToColor(img.BasePixelsForTesting(), 0, 0)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShaderChain(t *testing.T) {
	if !graphicscommand.IsShaderAvailable() {
		t.Skip("shader is not available on this environment")
	}

	const num = 10
	imgs := []*Image{}
	for i := 0; i < num; i++ {
		img := NewImage(1, 1, false)
		defer img.Dispose()
		imgs = append(imgs, img)
	}

	imgs[0].ReplacePixels([]byte{0xff, 0, 0, 0xff}, 0, 0, 1, 1)

	ir := etesting.ShaderProgramImages(1)
	s := NewShader(&ir)
	for i := 0; i < num-1; i++ {
		us := []interface{}{
			[]float32{0, 0},
			imgs[i],
		}
		imgs[i+1].DrawTriangles(nil, quadVertices(1, 1, 0, 0), graphics.QuadIndices(), nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, s, us)
	}

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := RestoreIfNeeded(); err != nil {
		t.Fatal(err)
	}

	for i, img := range imgs {
		want := color.RGBA{0xff, 0, 0, 0xff}
		got := pixelsToColor(img.BasePixelsForTesting(), 0, 0)
		if !sameColors(got, want, 1) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestShaderMultipleSources(t *testing.T) {
	if !graphicscommand.IsShaderAvailable() {
		t.Skip("shader is not available on this environment")
	}

	srcs := make([]*Image, 3)
	for i := range srcs {
		srcs[i] = NewImage(1, 1, false)
	}
	srcs[0].ReplacePixels([]byte{0x40, 0, 0, 0xff}, 0, 0, 1, 1)
	srcs[1].ReplacePixels([]byte{0, 0x80, 0, 0xff}, 0, 0, 1, 1)
	srcs[2].ReplacePixels([]byte{0, 0, 0xc0, 0xff}, 0, 0, 1, 1)

	dst := NewImage(1, 1, false)

	ir := etesting.ShaderProgramImages(3)
	s := NewShader(&ir)
	us := []interface{}{
		[]float32{0, 0},
		srcs[0],
		srcs[1],
		srcs[2],
	}
	dst.DrawTriangles(nil, quadVertices(1, 1, 0, 0), graphics.QuadIndices(), nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, s, us)

	// Clear one of the sources after DrawTriangles. dst should not be affected.
	srcs[0].Fill(color.RGBA{})

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := RestoreIfNeeded(); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{0x40, 0x80, 0xc0, 0xff}
	got := pixelsToColor(dst.BasePixelsForTesting(), 0, 0)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShaderDispose(t *testing.T) {
	if !graphicscommand.IsShaderAvailable() {
		t.Skip("shader is not available on this environment")
	}

	img := NewImage(1, 1, false)
	defer img.Dispose()

	ir := etesting.ShaderProgramFill(0xff, 0, 0, 0xff)
	s := NewShader(&ir)
	us := []interface{}{
		[]float32{0, 0},
	}
	img.DrawTriangles(nil, quadVertices(1, 1, 0, 0), graphics.QuadIndices(), nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, s, us)

	// Dispose the shader. This should invalidates all the images using this shader i.e., all the images become
	// stale.
	s.Dispose()

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := RestoreIfNeeded(); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{0xff, 0, 0, 0xff}
	got := pixelsToColor(img.BasePixelsForTesting(), 0, 0)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}
