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
	"math"
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

func TestCircleVertexCount(t *testing.T) {
	tests := []struct {
		name   string
		radius float32
		want   int
	}{
		{
			name:   "negative",
			radius: -1,
		},
		{
			name: "zero",
		},
		{
			name:   "small",
			radius: 0.5,
			want:   2,
		},
		{
			name:   "ordinary",
			radius: 100,
			want:   315,
		},
		{
			name:   "huge",
			radius: 1e9,
			want:   8192,
		},
		{
			name:   "positive infinity",
			radius: float32(math.Inf(1)),
		},
		{
			name:   "negative infinity",
			radius: float32(math.Inf(-1)),
		},
		{
			name:   "NaN",
			radius: float32(math.NaN()),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := vector.CircleVertexCount(test.radius); got != test.want {
				t.Errorf("CircleVertexCount(%v): got %d, want %d", test.radius, got, test.want)
			}
		})
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

// nil options should be treated as the zero values, as FillPath does.
func TestStrokePathNilOptions(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	defer dst.Deallocate()

	var path vector.Path
	path.MoveTo(4, 4)
	path.LineTo(12, 12)
	vector.StrokePath(dst, &path, nil, nil)

	// A zero-width stroke renders nothing.
	if got, want := dst.At(8, 8), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestStrokePathKeepsSourcePath(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	defer dst.Deallocate()

	var path vector.Path
	path.MoveTo(1, 1)
	// A redundant line.
	path.LineTo(1, 1)
	path.LineTo(15, 1)
	// A cusp.
	path.QuadTo(15, 8, 15, 1)
	// A collinear curve.
	path.QuadTo(8, 1, 1, 1)
	path.Close()
	path.MoveTo(2, 2)
	// A single point.
	path.QuadTo(2, 2, 2, 2)

	// StrokePath must not modify the given path.
	want := vector.PathOperationsString(&path)
	vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: 2}, nil)
	if got := vector.PathOperationsString(&path); got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}
}
