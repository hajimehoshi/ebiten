// Copyright 2019 The Ebiten Authors
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

package buffered_test

import (
	"errors"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten"
)

var mainCh = make(chan func())

func runOnMainThread(f func()) {
	ch := make(chan struct{})
	mainCh <- func() {
		f()
		close(ch)
	}
	<-ch
}

var regularTermination = errors.New("regular termination")

type game struct {
	m     *testing.M
	endCh chan struct{}
	code  int
}

func (g *game) Update(*ebiten.Image) error {
	select {
	case f := <-mainCh:
		f()
	case <-g.endCh:
		return regularTermination
	}
	return nil
}

func (*game) Draw(*ebiten.Image) {
}

func (*game) Layout(int, int) (int, int) {
	return 320, 240
}

func TestMain(m *testing.M) {
	codeCh := make(chan int)
	endCh := make(chan struct{})
	go func() {
		code := m.Run()
		close(endCh)
		codeCh <- code
		close(codeCh)
	}()

	g := &game{
		m:     m,
		endCh: endCh,
	}
	if err := ebiten.RunGame(g); err != nil && err != regularTermination {
		panic(err)
	}

	os.Exit(<-codeCh)
}

type testResult struct {
	want color.RGBA
	got  <-chan color.RGBA
}

var testSetBeforeMainResult = func() testResult {
	clr := color.RGBA{1, 2, 3, 4}
	img, _ := ebiten.NewImage(16, 16, ebiten.FilterDefault)
	img.Set(0, 0, clr)

	ch := make(chan color.RGBA, 1)
	go func() {
		runOnMainThread(func() {
			ch <- img.At(0, 0).(color.RGBA)
		})
	}()

	return testResult{
		want: clr,
		got:  ch,
	}
}()

func TestSetBeforeMain(t *testing.T) {
	got := <-testSetBeforeMainResult.got
	want := testSetBeforeMainResult.want

	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

var testDrawImageBeforeMainResult = func() testResult {
	const w, h = 16, 16
	src, _ := ebiten.NewImage(w, h, ebiten.FilterDefault)
	dst, _ := ebiten.NewImage(w, h, ebiten.FilterDefault)
	src.Set(0, 0, color.White)
	dst.DrawImage(src, nil)

	ch := make(chan color.RGBA, 1)
	go func() {
		runOnMainThread(func() {
			ch <- dst.At(0, 0).(color.RGBA)
		})
	}()

	return testResult{
		want: color.RGBA{0xff, 0xff, 0xff, 0xff},
		got:  ch,
	}
}()

func TestDrawImageBeforeMain(t *testing.T) {
	got := <-testDrawImageBeforeMainResult.got
	want := testDrawImageBeforeMainResult.want

	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

var testDrawTrianglesBeforeMainResult = func() testResult {
	const w, h = 16, 16
	src, _ := ebiten.NewImage(w, h, ebiten.FilterDefault)
	dst, _ := ebiten.NewImage(w, h, ebiten.FilterDefault)
	src.Set(0, 0, color.White)
	vs := []ebiten.Vertex{
		{
			DstX:   0,
			DstY:   0,
			SrcX:   0,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   w,
			DstY:   0,
			SrcX:   w,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   0,
			DstY:   h,
			SrcX:   0,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	dst.DrawTriangles(vs, []uint16{0, 1, 2}, src, nil)

	ch := make(chan color.RGBA, 1)
	go func() {
		runOnMainThread(func() {
			ch <- dst.At(0, 0).(color.RGBA)
		})
	}()

	return testResult{
		want: color.RGBA{0xff, 0xff, 0xff, 0xff},
		got:  ch,
	}
}()

func TestDrawTrianglesBeforeMain(t *testing.T) {
	got := <-testDrawTrianglesBeforeMainResult.got
	want := testDrawTrianglesBeforeMainResult.want

	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

var testSetAndFillBeforeMainResult = func() testResult {
	clr := color.RGBA{1, 2, 3, 4}
	img, _ := ebiten.NewImage(16, 16, ebiten.FilterDefault)
	img.Set(0, 0, clr)
	img.Fill(color.RGBA{5, 6, 7, 8})
	img.Set(1, 0, clr)

	ch := make(chan color.RGBA, 1)
	go func() {
		runOnMainThread(func() {
			ch <- img.At(0, 0).(color.RGBA)
		})
	}()

	return testResult{
		want: color.RGBA{5, 6, 7, 8},
		got:  ch,
	}
}()

func TestSetAndFillBeforeMain(t *testing.T) {
	got := <-testSetAndFillBeforeMainResult.got
	want := testSetAndFillBeforeMainResult.want

	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

var testSetAndReplacePixelsBeforeMainResult = func() testResult {
	clr := color.RGBA{1, 2, 3, 4}
	img, _ := ebiten.NewImage(16, 16, ebiten.FilterDefault)
	img.Set(0, 0, clr)
	pix := make([]byte, 4*16*16)
	for i := 0; i < len(pix)/4; i++ {
		pix[4*i] = 5
		pix[4*i+1] = 6
		pix[4*i+2] = 7
		pix[4*i+3] = 8
	}
	img.ReplacePixels(pix)
	img.Set(1, 0, clr)

	ch := make(chan color.RGBA, 1)
	go func() {
		runOnMainThread(func() {
			ch <- img.At(0, 0).(color.RGBA)
		})
	}()

	return testResult{
		want: color.RGBA{5, 6, 7, 8},
		got:  ch,
	}
}()

func TestSetAndReplacePixelsBeforeMain(t *testing.T) {
	got := <-testSetAndReplacePixelsBeforeMainResult.got
	want := testSetAndReplacePixelsBeforeMainResult.want

	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

var testReplacePixelsAndModifyBeforeMainResult = func() testResult {
	img, _ := ebiten.NewImage(16, 16, ebiten.FilterDefault)
	pix := make([]byte, 4*16*16)
	for i := 0; i < len(pix)/4; i++ {
		pix[4*i] = 1
		pix[4*i+1] = 2
		pix[4*i+2] = 3
		pix[4*i+3] = 4
	}
	img.ReplacePixels(pix)
	// After calling ReplacePixels, modifying pix must not affect the result.
	for i := 0; i < len(pix)/4; i++ {
		pix[4*i] = 5
		pix[4*i+1] = 6
		pix[4*i+2] = 7
		pix[4*i+3] = 8
	}

	ch := make(chan color.RGBA, 1)
	go func() {
		runOnMainThread(func() {
			ch <- img.At(0, 0).(color.RGBA)
		})
	}()

	return testResult{
		want: color.RGBA{1, 2, 3, 4},
		got:  ch,
	}
}()

func TestReplacePixelsAndModifyBeforeMain(t *testing.T) {
	got := <-testReplacePixelsAndModifyBeforeMainResult.got
	want := testReplacePixelsAndModifyBeforeMainResult.want

	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TODO: Add tests for shaders and ReplacePixels to check resolvePendingPiexles works correctly.
