// Copyright 2023 The Ebitengine Authors
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

package vector_test

import (
	"image"
	"image/color"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestMain(m *testing.M) {
	t.MainWithRunLoop(m)
}

// Issue #2589
func TestLine0(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	vector.StrokeLine(dst, 0, 0, 0, 0, 2, color.White, true)
	if got, want := dst.At(0, 0), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #3270
func TestStrokeRectAntiAlias(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	vector.StrokeRect(dst, 0, 0, 16, 16, 2, color.White, true)
	if got, want := dst.At(5, 5), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #3330
func TestFillRectSubImage(t *testing.T) {
	dst := ebiten.NewImage(16, 16)

	dst2 := dst.SubImage(image.Rect(0, 0, 8, 8)).(*ebiten.Image)
	vector.FillRect(dst2, 0, 0, 8, 8, color.White, true)
	if got, want := dst.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	dst3 := dst2.SubImage(image.Rect(4, 4, 8, 8)).(*ebiten.Image)
	vector.FillRect(dst3, 4, 4, 4, 4, color.Black, true)
	if got, want := dst.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst3.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #3330
func TestFillCircleSubImage(t *testing.T) {
	dst := ebiten.NewImage(16, 16)

	dst2 := dst.SubImage(image.Rect(0, 0, 8, 8)).(*ebiten.Image)
	vector.FillCircle(dst2, 4, 4, 4, color.White, true)
	if got, want := dst.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	dst3 := dst2.SubImage(image.Rect(4, 4, 8, 8)).(*ebiten.Image)
	vector.FillCircle(dst3, 6, 6, 4, color.Black, true)
	if got, want := dst.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst3.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #3330
func TestFillPathSubImage(t *testing.T) {
	dst := ebiten.NewImage(16, 16)

	dst2 := dst.SubImage(image.Rect(0, 0, 8, 8)).(*ebiten.Image)
	var p vector.Path
	p.MoveTo(0, 0)
	p.LineTo(8, 0)
	p.LineTo(8, 8)
	p.LineTo(0, 8)
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(color.White)
	op.AntiAlias = true
	vector.FillPath(dst2, &p, nil, op)
	if got, want := dst.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	dst3 := dst2.SubImage(image.Rect(4, 4, 8, 8)).(*ebiten.Image)
	var p2 vector.Path
	p2.MoveTo(4, 4)
	p2.LineTo(8, 4)
	p2.LineTo(8, 8)
	p2.LineTo(4, 8)
	p2.Close()
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(color.Black)
	vector.FillPath(dst3, &p2, nil, op)
	if got, want := dst.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst3.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestRaceConditionWithSubImage(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)

	var wg sync.WaitGroup
	for i := range h {
		for j := range w {
			wg.Add(1)
			go func() {
				subImg := src.SubImage(image.Rect(i, j, i+1, j+1)).(*ebiten.Image)
				var p vector.Path
				p.MoveTo(0, 0)
				p.LineTo(w, 0)
				p.LineTo(w, h)
				p.LineTo(0, h)
				p.Close()
				op := &vector.DrawPathOptions{}
				op.ColorScale.ScaleWithColor(color.White)
				op.AntiAlias = true
				vector.FillPath(subImg, &p, nil, op)
				dst := ebiten.NewImage(w, h)
				dst.DrawImage(subImg, nil)
				wg.Done()
			}()
		}
	}
	wg.Wait()
}

// Issue #3355
func TestFillPathSubImageAndImage(t *testing.T) {
	dst := ebiten.NewImage(200, 200)
	defer dst.Deallocate()
	for i := range 100 {
		var path vector.Path
		path.LineTo(0, 0)
		path.LineTo(0, 100)
		path.LineTo(100, 100)
		path.LineTo(100, 0)
		path.LineTo(0, 0)
		path.Close()
		drawOp := &vector.DrawPathOptions{}
		drawOp.ColorScale.ScaleWithColor(color.RGBA{255, 0, 0, 255})
		subDst := dst.SubImage(image.Rect(0, 0, 100, 100)).(*ebiten.Image)
		vector.FillPath(subDst, &path, nil, drawOp)
		drawOp.ColorScale.Reset()
		drawOp.ColorScale.ScaleWithColor(color.RGBA{0, 255, 0, 255})
		vector.FillPath(dst, &path, nil, drawOp)

		if got, want := dst.At(50, 50), (color.RGBA{0, 255, 0, 255}); got != want {
			t.Errorf("%d: got: %v, want: %v", i, got, want)
		}
	}
}

// Issue #3357
func TestFillRects(t *testing.T) {
	dsts := []*ebiten.Image{
		ebiten.NewImage(1920, 1080),
		ebiten.NewImage(1920, 1080),
	}
	for _, dst := range dsts {
		defer dst.Deallocate()
	}

	for i, antialias := range []bool{true, false} {
		dst := dsts[i]
		vector.FillRect(dst, 593, -609, 1144, 1969, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
		vector.FillRect(dst, 613, -146, 1124, 446, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
		vector.FillRect(dst, 634, -80, 1103, 190, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
		vector.FillRect(dst, 634, 110, 1103, 190, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
		vector.FillRect(dst, 613, 300, 1124, 998, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
		vector.FillRect(dst, 634, 433, 1104, 865, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
		vector.FillRect(dst, 654, 495, 1084, 741, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
		vector.FillRect(dst, 674, 592, 1063, 644, color.RGBA{0x10, 0x00, 0x00, 0x10}, antialias)
	}

	got := dsts[0].At(800, 0)
	want := dsts[1].At(800, 0)
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #3377
func TestFillRectOnBigImage(t *testing.T) {
	dst := ebiten.NewImage(3000, 3000)
	defer dst.Deallocate()

	vector.FillRect(dst, 0, 0, 3000, 3000, color.White, true)
	if got, want := dst.At(0, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst.At(2980, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst.At(0, 2980), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst.At(2980, 2980), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
