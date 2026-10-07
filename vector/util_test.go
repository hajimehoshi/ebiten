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
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestMain(m *testing.M) {
	ui.SetPanicOnErrorOnReadingPixelsForTesting(true)
	ebiten.SetWindowVisible(false)
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
		radius float64
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
			name:   "maximum finite radius",
			radius: math.MaxFloat32,
			want:   8192,
		},
		{
			name:   "unrepresentable radius",
			radius: 1.5 * math.MaxFloat32,
		},
		{
			name:   "positive infinity",
			radius: math.Inf(1),
		},
		{
			name:   "negative infinity",
			radius: math.Inf(-1),
		},
		{
			name:   "NaN",
			radius: math.NaN(),
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

func TestStrokeCircleThickStrokeNonAntiAlias(t *testing.T) {
	dst := ebiten.NewImage(64, 64)
	defer dst.Deallocate()
	vector.StrokeCircle(dst, 32, 32, 10, 10, color.White, false)
	if got, want := dst.At(32, 32), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	dst2 := ebiten.NewImage(64, 64)
	defer dst2.Deallocate()
	vector.StrokeCircle(dst2, 32, 32, 10, 20, color.White, false)
	if got, want := dst2.At(32, 32), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestStrokeLineWideArithmetic(t *testing.T) {
	tests := []struct {
		name           string
		x0, y0, x1, y1 float32
	}{
		{
			name: "overflowing unequal differences",
			x0:   -math.MaxFloat32,
			y0:   -math.MaxFloat32,
			x1:   math.MaxFloat32,
			y1:   math.MaxFloat32 / 2,
		},
		{
			name: "rounded difference",
			x0:   -1,
			y0:   8,
			x1:   1 << 25,
			y1:   12,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			geoM := vector.StrokeLineGeoM(test.x0, test.y0, test.x1, test.y1, 2)
			// The cap centers must reach the endpoints to float64 precision
			// relative to the line length, including rotation error.
			for i, endpoint := range []struct {
				x, y float32
			}{
				{
					x: test.x0,
					y: test.y0,
				},
				{
					x: test.x1,
					y: test.y1,
				},
			} {
				x, y := geoM.Apply(float64(i), 0.5)
				for axis, pair := range [][2]float64{
					{x, float64(endpoint.x)},
					{y, float64(endpoint.y)},
				} {
					got, want := pair[0], pair[1]
					if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > 1e-14*math.Max(1, math.Hypot(float64(test.x1)-float64(test.x0), float64(test.y1)-float64(test.y0))) {
						t.Errorf("cap %d axis %d: got %g, want %g", i, axis, got, want)
					}
				}
			}
		})
	}
}

func TestUtilRectTranslated(t *testing.T) {
	const origin = 1 << 24
	clr := color.RGBA{R: 0x60, G: 0x30, B: 0x18, A: 0x80}
	tests := []struct {
		name string
		draw func(*ebiten.Image, float32, float32, bool)
	}{
		{
			name: "fill",
			draw: func(dst *ebiten.Image, x, y float32, aa bool) {
				vector.FillRect(dst, x, y, 9, 9, clr, aa)
			},
		},
		{
			name: "stroke",
			draw: func(dst *ebiten.Image, x, y float32, aa bool) {
				vector.StrokeRect(dst, x, y, 9, 9, 4, clr, aa)
			},
		},
		{
			name: "thick stroke",
			draw: func(dst *ebiten.Image, x, y float32, aa bool) {
				vector.StrokeRect(dst, x, y, 3, 9, 4, clr, aa)
			},
		},
	}
	for _, test := range tests {
		for _, aa := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/antialias=%t", test.name, aa), func(t *testing.T) {
				want := ebiten.NewImage(16, 16)
				defer want.Deallocate()
				got := ebiten.NewImageWithOptions(image.Rect(origin, origin, origin+16, origin+16), nil)
				defer got.Deallocate()
				test.draw(want, 4, 4, aa)
				test.draw(got, origin+4, origin+4, aa)
				wantPixels := make([]byte, 16*16*4)
				gotPixels := make([]byte, len(wantPixels))
				want.ReadPixels(wantPixels)
				got.ReadPixels(gotPixels)
				for i, value := range gotPixels {
					if value != wantPixels[i] {
						t.Errorf("pixel (%d, %d), channel %d: got %d, want %d", i/4%16, i/4/16, i%4, value, wantPixels[i])
						break
					}
				}
			})
		}
	}
}

func TestUtilCircleWideVertices(t *testing.T) {
	tests := []struct {
		name                  string
		cx, cy, radius, width float32
		stroke                bool
	}{
		{
			name:   "fill trigonometry and cancellation",
			cx:     -1e7,
			cy:     -1e7,
			radius: 1e7,
		},
		{
			name:   "stroke radii and cancellation",
			cx:     -(1 << 25),
			cy:     -(1 << 25),
			radius: 1 << 25,
			width:  2,
			stroke: true,
		},
		{
			name:   "thick stroke radius",
			cx:     -(1 << 25),
			cy:     -(1 << 25),
			radius: 1,
			width:  1 << 26,
			stroke: true,
		},
		{
			name:   "stroke tessellation threshold",
			radius: 1.4323945,
			width:  0.95492965,
			stroke: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dst := ebiten.NewImage(16, 16)
			defer dst.Deallocate()
			vertices := vector.UtilDrawingVertices(func() {
				if test.stroke {
					vector.StrokeCircle(dst, test.cx, test.cy, test.radius, test.width, color.White, false)
				} else {
					vector.FillCircle(dst, test.cx, test.cy, test.radius, color.White, false)
				}
			})
			outerRadius := float64(test.radius)
			if test.stroke {
				outerRadius += float64(test.width) / 2
			}
			count := min(8192, int(math.Ceil(math.Pi*outerRadius)))
			ring := test.stroke && float64(test.width) < 2*float64(test.radius)
			wantCount := count
			if ring {
				wantCount *= 2
			}
			if len(vertices) != wantCount {
				t.Fatalf("vertex count: got %d, want %d", len(vertices), wantCount)
			}
			for i, v := range vertices {
				angleIndex := i
				radius := outerRadius
				if ring {
					angleIndex /= 2
					if i%2 != 0 {
						radius = float64(test.radius) - float64(test.width)/2
					}
				}
				angle := float64(angleIndex) * (2 * math.Pi / float64(count))
				wantX := float32(float64(test.cx) + radius*math.Cos(angle))
				wantY := float32(float64(test.cy) + radius*math.Sin(angle))
				if v.DstX != wantX || v.DstY != wantY {
					t.Errorf("vertex %d: got (%g, %g), want (%g, %g)", i, v.DstX, v.DstY, wantX, wantY)
					break
				}
			}
		})
	}
}

func TestUtilOrdinaryDrawing(t *testing.T) {
	clr := color.RGBA{R: 0x60, G: 0x30, B: 0x18, A: 0x80}
	tests := []struct {
		name            string
		draw            func(*ebiten.Image, bool)
		inside, outside image.Point
	}{
		{
			name: "line",
			draw: func(dst *ebiten.Image, aa bool) {
				vector.StrokeLine(dst, 4, 8, 12, 8, 4, clr, aa)
			},
			inside:  image.Point{X: 8, Y: 8},
			outside: image.Point{X: 8, Y: 4},
		},
		{
			name: "circle",
			draw: func(dst *ebiten.Image, aa bool) {
				vector.FillCircle(dst, 8, 8, 4, clr, aa)
			},
			inside:  image.Point{X: 8, Y: 8},
			outside: image.Point{X: 2, Y: 8},
		},
		{
			name: "circle stroke",
			draw: func(dst *ebiten.Image, aa bool) {
				vector.StrokeCircle(dst, 8, 8, 5, 4, clr, aa)
			},
			inside:  image.Point{X: 12, Y: 8},
			outside: image.Point{X: 8, Y: 8},
		},
	}
	for _, test := range tests {
		for _, aa := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/antialias=%t", test.name, aa), func(t *testing.T) {
				dst := ebiten.NewImage(16, 16)
				defer dst.Deallocate()
				test.draw(dst, aa)
				if got := dst.At(test.inside.X, test.inside.Y); got != clr {
					t.Errorf("inside: got %v, want %v", got, clr)
				}
				if got, want := dst.At(test.outside.X, test.outside.Y), (color.RGBA{}); got != want {
					t.Errorf("outside: got %v, want %v", got, want)
				}
			})
		}
	}
}

func TestUtilUnrepresentableGeometry(t *testing.T) {
	tests := []struct {
		name string
		draw func(*ebiten.Image)
	}{
		{
			name: "antialiased rectangle edge",
			draw: func(dst *ebiten.Image) {
				vector.FillRect(dst, math.MaxFloat32, 0, math.MaxFloat32, 16, color.White, true)
			},
		},
		{
			name: "overflowing circle outer radius",
			draw: func(dst *ebiten.Image) {
				vector.StrokeCircle(dst, 8, 8, math.MaxFloat32, math.MaxFloat32, color.White, false)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dst := ebiten.NewImage(16, 16)
			defer dst.Deallocate()
			test.draw(dst)
			pixels := make([]byte, 16*16*4)
			dst.ReadPixels(pixels)
			for i, value := range pixels {
				if value != 0 {
					t.Errorf("byte %d: got %d, want 0", i, value)
					break
				}
			}
		})
	}
}

func TestStrokeRectThickStrokeWideOffset(t *testing.T) {
	const origin = 1 << 24
	dst := ebiten.NewImageWithOptions(image.Rect(origin, origin, origin+16, origin+16), nil)
	defer dst.Deallocate()
	// The expanded rectangle starts three pixels before the input origin.
	vector.StrokeRect(dst, origin+4, origin+4, 3, 9, 6, color.White, false)
	pixels := make([]byte, 16*16*4)
	dst.ReadPixels(pixels)
	for y := range 16 {
		for x := range 16 {
			var want byte
			if x >= 1 && x < 10 && y >= 1 {
				want = 0xff
			}
			for channel := range 4 {
				if got := pixels[4*(16*y+x)+channel]; got != want {
					t.Errorf("pixel (%d, %d), channel %d: got %d, want %d", x, y, channel, got, want)
					return
				}
			}
		}
	}
}
