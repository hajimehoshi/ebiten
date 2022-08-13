// Copyright 2016 Hajime Hoshi
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

package ebiten_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// maxImageSize is a maximum image size that should work in almost every environment.
const maxImageSize = 4096 - 2

func init() {
	rand.Seed(time.Now().UnixNano())
}

func skipTooSlowTests(t *testing.T) bool {
	if testing.Short() {
		t.Skip("skipping test in short mode")
		return true
	}
	if runtime.GOOS == "js" {
		t.Skip("too slow or fragile on Wasm")
		return true
	}
	return false
}

func TestMain(m *testing.M) {
	ui.SetPanicOnErrorOnReadingPixelsForTesting(true)
	t.MainWithRunLoop(m)
}

func openEbitenImage() (*ebiten.Image, image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		return nil, nil, err
	}

	eimg := ebiten.NewImageFromImage(img)
	return eimg, img, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// sameColors compares c1 and c2 and returns a boolean value indicating
// if the two colors are (almost) same.
//
// Pixels read from GPU might include errors (#492), and
// sameColors considers such errors as delta.
func sameColors(c1, c2 color.RGBA, delta int) bool {
	return abs(int(c1.R)-int(c2.R)) <= delta &&
		abs(int(c1.G)-int(c2.G)) <= delta &&
		abs(int(c1.B)-int(c2.B)) <= delta &&
		abs(int(c1.A)-int(c2.A)) <= delta
}

func TestImagePixels(t *testing.T) {
	img0, img, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}

	if got := img0.Bounds().Size(); got != img.Bounds().Size() {
		t.Fatalf("img size: got %d; want %d", got, img.Bounds().Size())
	}

	w, h := img0.Bounds().Size().X, img0.Bounds().Size().Y
	// Check out of range part
	w2, h2 := graphics.InternalImageSize(w), graphics.InternalImageSize(h)
	for j := -100; j < h2+100; j++ {
		for i := -100; i < w2+100; i++ {
			got := img0.At(i, j)
			want := color.RGBAModel.Convert(img.At(i, j))
			if got != want {
				t.Errorf("img0 At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}

	pix := make([]byte, 4*w*h)
	img0.ReadPixels(pix)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (j*w + i)
			got := color.RGBA{pix[idx], pix[idx+1], pix[idx+2], pix[idx+3]}
			want := color.RGBAModel.Convert(img.At(i, j))
			if got != want {
				t.Errorf("(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
}

func TestImageComposition(t *testing.T) {
	img2Color := color.NRGBA{0x24, 0x3f, 0x6a, 0x88}
	img3Color := color.NRGBA{0x85, 0xa3, 0x08, 0xd3}

	// TODO: Rename this to img0
	img1, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}

	w, h := img1.Bounds().Size().X, img1.Bounds().Size().Y

	img2 := ebiten.NewImage(w, h)
	img3 := ebiten.NewImage(w, h)

	img2.Fill(img2Color)
	img3.Fill(img3Color)
	img_12_3 := ebiten.NewImage(w, h)
	img2.DrawImage(img1, nil)
	img3.DrawImage(img2, nil)
	img_12_3.DrawImage(img3, nil)

	img2.Fill(img2Color)
	img3.Fill(img3Color)
	img_1_23 := ebiten.NewImage(w, h)
	img3.DrawImage(img2, nil)
	img3.DrawImage(img1, nil)
	img_1_23.DrawImage(img3, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c1 := img_12_3.At(i, j).(color.RGBA)
			c2 := img_1_23.At(i, j).(color.RGBA)
			if !sameColors(c1, c2, 1) {
				t.Errorf("img_12_3.At(%d, %d) = %v; img_1_23.At(%[1]d, %[2]d) = %#[4]v", i, j, c1, c2)
			}
			if c1.A == 0 {
				t.Fatalf("img_12_3.At(%d, %d).A = 0; nothing is rendered?", i, j)
			}
			if c2.A == 0 {
				t.Fatalf("img_1_23.At(%d, %d).A = 0; nothing is rendered?", i, j)
			}
		}
	}
}

func TestImageSelf(t *testing.T) {
	// Note that mutex usages: without defer, unlocking is not called when panicing.
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DrawImage must panic but not")
		}
	}()
	img, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	img.DrawImage(img, nil)
}

func TestImageScale(t *testing.T) {
	for _, scale := range []int{2, 3, 4} {
		img0, _, err := openEbitenImage()
		if err != nil {
			t.Fatal(err)
			return
		}
		w, h := img0.Size()
		img1 := ebiten.NewImage(w*scale, h*scale)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(scale), float64(scale))

		img1.DrawImage(img0, op)

		for j := 0; j < h*scale; j++ {
			for i := 0; i < w*scale; i++ {
				c0 := img0.At(i/scale, j/scale).(color.RGBA)
				c1 := img1.At(i, j).(color.RGBA)
				if c0 != c1 {
					t.Fatalf("img0.At(%[1]d, %[2]d) should equal to img1.At(%[3]d, %[4]d) (with scale %[5]d) but not: %[6]v vs %[7]v", i/2, j/2, i, j, scale, c0, c1)
				}
			}
		}
	}
}

func TestImage90DegreeRotate(t *testing.T) {
	img0, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := img0.Size()
	img1 := ebiten.NewImage(h, w)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Rotate(math.Pi / 2)
	op.GeoM.Translate(float64(h), 0)
	img1.DrawImage(img0, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c0 := img0.At(i, j).(color.RGBA)
			c1 := img1.At(h-j-1, i).(color.RGBA)
			if c0 != c1 {
				t.Errorf("img0.At(%[1]d, %[2]d) should equal to img1.At(%[3]d, %[4]d) but not: %[5]v vs %[6]v", i, j, h-j-1, i, c0, c1)
			}
		}
	}
}

func TestImageDotByDotInversion(t *testing.T) {
	img0, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := img0.Size()
	img1 := ebiten.NewImage(w, h)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Rotate(math.Pi)
	op.GeoM.Translate(float64(w), float64(h))
	img1.DrawImage(img0, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c0 := img0.At(i, j).(color.RGBA)
			c1 := img1.At(w-i-1, h-j-1).(color.RGBA)
			if c0 != c1 {
				t.Errorf("img0.At(%[1]d, %[2]d) should equal to img1.At(%[3]d, %[4]d) but not: %[5]v vs %[6]v", i, j, w-i-1, h-j-1, c0, c1)
			}
		}
	}
}

func TestImageWritePixels(t *testing.T) {
	// Create a dummy image so that the shared texture is used and origImg's position is shfited.
	dummyImg := ebiten.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 16, 16)))
	defer dummyImg.Dispose()

	_, origImg, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	// Convert to *image.RGBA just in case.
	img := image.NewRGBA(origImg.Bounds())
	draw.Draw(img, img.Bounds(), origImg, image.ZP, draw.Src)

	size := img.Bounds().Size()
	img0 := ebiten.NewImage(size.X, size.Y)

	img0.WritePixels(img.Pix)
	for j := 0; j < img0.Bounds().Size().Y; j++ {
		for i := 0; i < img0.Bounds().Size().X; i++ {
			got := img0.At(i, j)
			want := img.At(i, j)
			if got != want {
				t.Errorf("img0 At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}

	p := make([]uint8, 4*size.X*size.Y)
	for i := range p {
		p[i] = 0x80
	}
	img0.WritePixels(p)
	// Even if p is changed after calling ReplacePixel, img0 uses the original values.
	for i := range p {
		p[i] = 0
	}
	for j := 0; j < img0.Bounds().Size().Y; j++ {
		for i := 0; i < img0.Bounds().Size().X; i++ {
			got := img0.At(i, j)
			want := color.RGBA{0x80, 0x80, 0x80, 0x80}
			if got != want {
				t.Errorf("img0 At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
}

func TestImageWritePixelsNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("WritePixels(nil) must panic")
		}
	}()

	img := ebiten.NewImage(16, 16)
	img.Fill(color.White)
	img.WritePixels(nil)
}

func TestImageDispose(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Fill(color.White)
	img.Dispose()

	// The color is transparent (color.RGBA{}).
	// Note that the value's type must be color.RGBA.
	got := img.At(0, 0)
	want := color.RGBA{}
	if got != want {
		t.Errorf("img.At(0, 0) got: %v, want: %v", got, want)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestImageCompositeModeLighter(t *testing.T) {
	img0, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}

	w, h := img0.Size()
	img1 := ebiten.NewImage(w, h)
	img1.Fill(color.RGBA{0x01, 0x02, 0x03, 0x04})
	op := &ebiten.DrawImageOptions{}
	op.CompositeMode = ebiten.CompositeModeLighter
	img1.DrawImage(img0, op)
	for j := 0; j < img1.Bounds().Size().Y; j++ {
		for i := 0; i < img1.Bounds().Size().X; i++ {
			got := img1.At(i, j).(color.RGBA)
			want := img0.At(i, j).(color.RGBA)
			want.R = uint8(min(0xff, int(want.R)+1))
			want.G = uint8(min(0xff, int(want.G)+2))
			want.B = uint8(min(0xff, int(want.B)+3))
			want.A = uint8(min(0xff, int(want.A)+4))
			if got != want {
				t.Errorf("img1 At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
}

func TestNewImageFromEbitenImage(t *testing.T) {
	img, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	_ = ebiten.NewImageFromImage(img)
}

func TestNewImageFromSubImage(t *testing.T) {
	_, img, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	subImg := img.(*image.NRGBA).SubImage(image.Rect(1, 1, w-1, h-1))
	eimg := ebiten.NewImageFromImage(subImg)
	sw, sh := subImg.Bounds().Dx(), subImg.Bounds().Dy()
	w2, h2 := eimg.Size()
	if w2 != sw {
		t.Errorf("eimg Width: got %v; want %v", w2, sw)
	}
	if h2 != sh {
		t.Errorf("eimg Width: got %v; want %v", h2, sh)
	}
	for j := 0; j < h2; j++ {
		for i := 0; i < w2; i++ {
			got := eimg.At(i, j)
			want := color.RGBAModel.Convert(img.At(i+1, j+1))
			if got != want {
				t.Errorf("img0 At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
}

type mutableRGBA struct {
	r, g, b, a uint8
}

func (c *mutableRGBA) RGBA() (r, g, b, a uint32) {
	return uint32(c.r) * 0x101, uint32(c.g) * 0x101, uint32(c.b) * 0x101, uint32(c.a) * 0x101
}

func TestImageFill(t *testing.T) {
	w, h := 10, 10
	img := ebiten.NewImage(w, h)
	clr := &mutableRGBA{0x80, 0x80, 0x80, 0x80}
	img.Fill(clr)
	clr.r = 0
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want := color.RGBA{0x80, 0x80, 0x80, 0x80}
			if got != want {
				t.Errorf("img At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
}

// Issue #740
func TestImageClear(t *testing.T) {
	const w, h = 128, 256
	img := ebiten.NewImage(w, h)
	img.Fill(color.White)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("img At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
	img.Clear()
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want := color.RGBA{}
			if got != want {
				t.Errorf("img At(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
}

// Issue #317, #558, #724
func TestImageEdge(t *testing.T) {
	// TODO: This test is not so meaningful after #1218. Do we remove this?

	if skipTooSlowTests(t) {
		return
	}

	const (
		img0Width       = 10
		img0Height      = 10
		img0InnerWidth  = 10
		img0InnerHeight = 10

		img1Width  = 32
		img1Height = 32
	)
	img0 := ebiten.NewImage(img0Width, img0Height)
	pixels := make([]uint8, 4*img0Width*img0Height)
	for j := 0; j < img0Height; j++ {
		for i := 0; i < img0Width; i++ {
			idx := 4 * (i + j*img0Width)
			pixels[idx] = 0xff
			pixels[idx+1] = 0
			pixels[idx+2] = 0
			pixels[idx+3] = 0xff
		}
	}
	img0.WritePixels(pixels)
	img1 := ebiten.NewImage(img1Width, img1Height)
	red := color.RGBA{0xff, 0, 0, 0xff}
	transparent := color.RGBA{0, 0, 0, 0}

	angles := []float64{}
	for a := 0; a < 1440; a++ {
		angles = append(angles, float64(a)/1440*2*math.Pi)
	}
	for a := 0; a < 4096; a += 3 {
		// a++ should be fine, but it takes long to test.
		angles = append(angles, float64(a)/4096*2*math.Pi)
	}

	for _, s := range []float64{1, 0.5, 0.25} {
		for _, f := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
			for _, a := range angles {
				for _, testDrawTriangles := range []bool{false, true} {
					img1.Clear()
					w, h := img0.Size()
					b := img0.Bounds()
					var geo ebiten.GeoM
					geo.Translate(-float64(w)/2, -float64(h)/2)
					geo.Scale(s, s)
					geo.Rotate(a)
					geo.Translate(img1Width/2, img1Height/2)
					if !testDrawTriangles {
						op := &ebiten.DrawImageOptions{}
						op.GeoM = geo
						op.Filter = f
						img1.DrawImage(img0, op)
					} else {
						op := &ebiten.DrawTrianglesOptions{}
						dx0, dy0 := geo.Apply(0, 0)
						dx1, dy1 := geo.Apply(float64(w), 0)
						dx2, dy2 := geo.Apply(0, float64(h))
						dx3, dy3 := geo.Apply(float64(w), float64(h))
						vs := []ebiten.Vertex{
							{
								DstX:   float32(dx0),
								DstY:   float32(dy0),
								SrcX:   float32(b.Min.X),
								SrcY:   float32(b.Min.Y),
								ColorR: 1,
								ColorG: 1,
								ColorB: 1,
								ColorA: 1,
							},
							{
								DstX:   float32(dx1),
								DstY:   float32(dy1),
								SrcX:   float32(b.Max.X),
								SrcY:   float32(b.Min.Y),
								ColorR: 1,
								ColorG: 1,
								ColorB: 1,
								ColorA: 1,
							},
							{
								DstX:   float32(dx2),
								DstY:   float32(dy2),
								SrcX:   float32(b.Min.X),
								SrcY:   float32(b.Max.Y),
								ColorR: 1,
								ColorG: 1,
								ColorB: 1,
								ColorA: 1,
							},
							{
								DstX:   float32(dx3),
								DstY:   float32(dy3),
								SrcX:   float32(b.Max.X),
								SrcY:   float32(b.Max.Y),
								ColorR: 1,
								ColorG: 1,
								ColorB: 1,
								ColorA: 1,
							},
						}
						is := graphics.QuadIndices()
						op.Filter = f
						img1.DrawTriangles(vs, is, img0, op)
					}
					allTransparent := true
					for j := 0; j < img1Height; j++ {
						for i := 0; i < img1Width; i++ {
							c := img1.At(i, j)
							if c == transparent {
								continue
							}
							allTransparent = false
							switch f {
							case ebiten.FilterNearest:
								if c == red {
									continue
								}
							case ebiten.FilterLinear:
								if _, g, b, _ := c.RGBA(); g == 0 && b == 0 {
									continue
								}
							}
							t.Fatalf("img1.At(%d, %d) (filter: %d, scale: %f, angle: %f, draw-triangles?: %t) want: red or transparent, got: %v", i, j, f, s, a, testDrawTriangles, c)
						}
					}
					if allTransparent {
						t.Fatalf("img1 (filter: %d, scale: %f, angle: %f, draw-triangles?: %t) is transparent but should not", f, s, a, testDrawTriangles)
					}
				}
			}
		}
	}
}

// Issue #419
func TestImageTooManyFill(t *testing.T) {
	const width = 1024

	indexToColor := func(index int) uint8 {
		return uint8((17*index + 0x40) % 256)
	}

	src := ebiten.NewImage(1, 1)
	dst := ebiten.NewImage(width, 1)
	for i := 0; i < width; i++ {
		c := indexToColor(i)
		src.Fill(color.RGBA{c, c, c, 0xff})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(i), 0)
		dst.DrawImage(src, op)
	}

	for i := 0; i < width; i++ {
		c := indexToColor(i)
		got := dst.At(i, 0).(color.RGBA)
		want := color.RGBA{c, c, c, 0xff}
		if !sameColors(got, want, 1) {
			t.Errorf("dst.At(%d, %d): got %v, want: %v", i, 0, got, want)
		}
	}
}

func BenchmarkDrawImage(b *testing.B) {
	img0 := ebiten.NewImage(16, 16)
	img1 := ebiten.NewImage(16, 16)
	op := &ebiten.DrawImageOptions{}
	for i := 0; i < b.N; i++ {
		img0.DrawImage(img1, op)
	}
}

func TestImageLinearGraduation(t *testing.T) {
	img0 := ebiten.NewImage(2, 2)
	img0.WritePixels([]byte{
		0xff, 0x00, 0x00, 0xff,
		0x00, 0xff, 0x00, 0xff,
		0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	})

	const w, h = 32, 32
	img1 := ebiten.NewImage(w, h)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(-w/4, -h/4)
	op.Filter = ebiten.FilterLinear
	img1.DrawImage(img0, op)

	for j := 1; j < h-1; j++ {
		for i := 1; i < w-1; i++ {
			c := img1.At(i, j).(color.RGBA)
			if c.R == 0 || c.R == 0xff {
				t.Errorf("img1.At(%d, %d).R must be in between 0x01 and 0xfe but %v", i, j, c)
			}
		}
	}
}

func TestImageOutside(t *testing.T) {
	src := ebiten.NewImage(5, 10) // internal texture size is 8x16.
	dst := ebiten.NewImage(4, 4)
	src.Fill(color.RGBA{0xff, 0, 0, 0xff})

	cases := []struct {
		X, Y, Width, Height int
	}{
		{-4, -4, 4, 4},
		{5, 0, 4, 4},
		{0, 10, 4, 4},
		{5, 10, 4, 4},
		{8, 0, 4, 4},
		{0, 16, 4, 4},
		{8, 16, 4, 4},
		{8, -4, 4, 4},
		{-4, 16, 4, 4},
		{5, 10, 0, 0},
		{5, 10, -2, -2}, // non-well-formed rectangle
	}
	for _, c := range cases {
		dst.Clear()

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, 0)
		dst.DrawImage(src.SubImage(image.Rectangle{
			Min: image.Pt(c.X, c.Y),
			Max: image.Pt(c.X+c.Width, c.Y+c.Height),
		}).(*ebiten.Image), op)

		for j := 0; j < 4; j++ {
			for i := 0; i < 4; i++ {
				got := dst.At(i, j).(color.RGBA)
				want := color.RGBA{0, 0, 0, 0}
				if got != want {
					t.Errorf("src(x: %d, y: %d, w: %d, h: %d), dst At(%d, %d): got %v, want: %v", c.X, c.Y, c.Width, c.Height, i, j, got, want)
				}
			}
		}
	}
}

func TestImageOutsideUpperLeft(t *testing.T) {
	src := ebiten.NewImage(4, 4)
	dst1 := ebiten.NewImage(16, 16)
	dst2 := ebiten.NewImage(16, 16)
	src.Fill(color.RGBA{0xff, 0, 0, 0xff})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Rotate(math.Pi / 4)
	dst1.DrawImage(src.SubImage(image.Rect(-4, -4, 8, 8)).(*ebiten.Image), op)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Rotate(math.Pi / 4)
	dst2.DrawImage(src, op)

	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			got := dst1.At(i, j).(color.RGBA)
			want := dst2.At(i, j).(color.RGBA)
			if got != want {
				t.Errorf("got: dst1.At(%d, %d): %v, want: dst2.At(%d, %d): %v", i, j, got, i, j, want)
			}
		}
	}
}

func TestImageSize(t *testing.T) {
	const (
		w = 17
		h = 31
	)
	img := ebiten.NewImage(w, h)
	gotW, gotH := img.Size()
	if gotW != w {
		t.Errorf("got: %d, want: %d", gotW, w)
	}
	if gotH != h {
		t.Errorf("got: %d, want: %d", gotH, h)
	}
}

func TestImageSize1(t *testing.T) {
	src := ebiten.NewImage(1, 1)
	dst := ebiten.NewImage(1, 1)
	src.Fill(color.White)
	dst.DrawImage(src, nil)
	got := src.At(0, 0).(color.RGBA)
	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	if !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TODO: Enable this test again. This test fails after #1217 is fixed.
func Skip_TestImageSize4096(t *testing.T) {
	src := ebiten.NewImage(4096, 4096)
	dst := ebiten.NewImage(4096, 4096)
	pix := make([]byte, 4096*4096*4)
	for i := 0; i < 4096; i++ {
		j := 4095
		idx := 4 * (i + j*4096)
		pix[idx] = uint8(i + j)
		pix[idx+1] = uint8((i + j) >> 8)
		pix[idx+2] = uint8((i + j) >> 16)
		pix[idx+3] = 0xff
	}
	for j := 0; j < 4096; j++ {
		i := 4095
		idx := 4 * (i + j*4096)
		pix[idx] = uint8(i + j)
		pix[idx+1] = uint8((i + j) >> 8)
		pix[idx+2] = uint8((i + j) >> 16)
		pix[idx+3] = 0xff
	}
	src.WritePixels(pix)
	dst.DrawImage(src, nil)
	for i := 4095; i < 4096; i++ {
		j := 4095
		got := dst.At(i, j).(color.RGBA)
		want := color.RGBA{uint8(i + j), uint8((i + j) >> 8), uint8((i + j) >> 16), 0xff}
		if got != want {
			t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
		}
	}
	for j := 4095; j < 4096; j++ {
		i := 4095
		got := dst.At(i, j).(color.RGBA)
		want := color.RGBA{uint8(i + j), uint8((i + j) >> 8), uint8((i + j) >> 16), 0xff}
		if got != want {
			t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
		}
	}
}

func TestImageCopy(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("copying image and using it must panic")
		}
	}()

	img0 := ebiten.NewImage(256, 256)
	img1 := *img0
	img1.Fill(color.Transparent)
}

// Issue #611, #907
func TestImageStretch(t *testing.T) {
	if skipTooSlowTests(t) {
		return
	}

	const w = 16

	dst := ebiten.NewImage(w, maxImageSize)
loop:
	for h := 1; h <= 32; h++ {
		src := ebiten.NewImage(w+2, h+2)

		pix := make([]byte, 4*(w+2)*(h+2))
		for i := 0; i < (w+2)*(h+2); i++ {
			pix[4*i] = 0xff
			pix[4*i+3] = 0xff
		}
		src.WritePixels(pix)

		_, dh := dst.Size()
		for i := 0; i < dh; {
			dst.Clear()
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(1, float64(i)/float64(h))
			dst.DrawImage(src.SubImage(image.Rect(1, 1, w+1, h+1)).(*ebiten.Image), op)
			for j := -1; j <= 1; j++ {
				if i+j < 0 {
					continue
				}
				got := dst.At(0, i+j).(color.RGBA)
				want := color.RGBA{}
				if j < 0 {
					want = color.RGBA{0xff, 0, 0, 0xff}
				}
				if got != want {
					t.Errorf("At(%d, %d) (height=%d, scale=%d/%d): got: %v, want: %v", 0, i+j, h, i, h, got, want)
					continue loop
				}
			}
			switch i % 32 {
			case 31, 0:
				i++
			case 1:
				i += 32 - 2
			default:
				panic("not reached")
			}
		}
	}
}

func TestImageSprites(t *testing.T) {
	const (
		width  = 512
		height = 512
	)

	src := ebiten.NewImage(4, 4)
	src.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	dst := ebiten.NewImage(width, height)
	for j := 0; j < height/4; j++ {
		for i := 0; i < width/4; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(i*4), float64(j*4))
			dst.DrawImage(src, op)
		}
	}

	for j := 0; j < height/4; j++ {
		for i := 0; i < width/4; i++ {
			got := dst.At(i*4, j*4).(color.RGBA)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got %v, want: %v", i*4, j*4, got, want)
			}
		}
	}
}

// Disabled: it does not make sense to expect deterministic mipmap results (#909).
func Disabled_TestImageMipmap(t *testing.T) {
	src, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := src.Size()

	l1 := ebiten.NewImage(w/2, h/2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = ebiten.FilterLinear
	l1.DrawImage(src, op)

	l1w, l1h := l1.Size()
	l2 := ebiten.NewImage(l1w/2, l1h/2)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = ebiten.FilterLinear
	l2.DrawImage(l1, op)

	gotDst := ebiten.NewImage(w, h)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/5.0, 1/5.0)
	op.Filter = ebiten.FilterLinear
	gotDst.DrawImage(src, op)

	wantDst := ebiten.NewImage(w, h)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4.0/5.0, 4.0/5.0)
	op.Filter = ebiten.FilterLinear
	wantDst.DrawImage(l2, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := gotDst.At(i, j).(color.RGBA)
			want := wantDst.At(i, j).(color.RGBA)
			if !sameColors(got, want, 1) {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Disabled: it does not make sense to expect deterministic mipmap results (#909).
func Disabled_TestImageMipmapNegativeDet(t *testing.T) {
	src, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := src.Size()

	l1 := ebiten.NewImage(w/2, h/2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = ebiten.FilterLinear
	l1.DrawImage(src, op)

	l1w, l1h := l1.Size()
	l2 := ebiten.NewImage(l1w/2, l1h/2)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = ebiten.FilterLinear
	l2.DrawImage(l1, op)

	gotDst := ebiten.NewImage(w, h)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(-1/5.0, -1/5.0)
	op.GeoM.Translate(float64(w), float64(h))
	op.Filter = ebiten.FilterLinear
	gotDst.DrawImage(src, op)

	wantDst := ebiten.NewImage(w, h)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(-4.0/5.0, -4.0/5.0)
	op.GeoM.Translate(float64(w), float64(h))
	op.Filter = ebiten.FilterLinear
	wantDst.DrawImage(l2, op)

	allZero := true
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := gotDst.At(i, j).(color.RGBA)
			want := wantDst.At(i, j).(color.RGBA)
			if !sameColors(got, want, 1) {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
			if got.A > 0 {
				allZero = false
			}
		}
	}

	if allZero {
		t.Errorf("the image must include non-zero values but not")
	}
}

// Issue #710
func TestImageMipmapColor(t *testing.T) {
	img0 := ebiten.NewImage(256, 256)
	img1 := ebiten.NewImage(128, 128)
	img1.Fill(color.White)

	for i := 0; i < 8; i++ {
		img0.Clear()

		s := 1 - float64(i)/8

		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterLinear
		op.GeoM.Scale(s, s)
		op.ColorM.Scale(1, 1, 0, 1)
		img0.DrawImage(img1, op)

		op.GeoM.Translate(128, 0)
		op.ColorM.Reset()
		op.ColorM.Scale(0, 1, 1, 1)
		img0.DrawImage(img1, op)

		want := color.RGBA{0, 0xff, 0xff, 0xff}
		got := img0.At(128, 0)
		if got != want {
			t.Errorf("want: %v, got: %v", want, got)
		}
	}
}

// Issue #725
func TestImageMiamapAndDrawTriangle(t *testing.T) {
	img0 := ebiten.NewImage(32, 32)
	img1 := ebiten.NewImage(128, 128)
	img2 := ebiten.NewImage(128, 128)

	// Fill img1 red and create img1's mipmap
	img1.Fill(color.RGBA{0xff, 0, 0, 0xff})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.25, 0.25)
	op.Filter = ebiten.FilterLinear
	img0.DrawImage(img1, op)

	// Call DrawTriangle on img1 and fill it with green
	img2.Fill(color.RGBA{0, 0xff, 0, 0xff})
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
			DstX:   128,
			DstY:   0,
			SrcX:   128,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   0,
			DstY:   128,
			SrcX:   0,
			SrcY:   128,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   128,
			DstY:   128,
			SrcX:   128,
			SrcY:   128,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	img1.DrawTriangles(vs, []uint16{0, 1, 2, 1, 2, 3}, img2, nil)

	// Draw img1 (green) again. Confirm mipmap is correctly updated.
	img0.Clear()
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.25, 0.25)
	op.Filter = ebiten.FilterLinear
	img0.DrawImage(img1, op)

	w, h := img0.Size()
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c := img0.At(i, j).(color.RGBA)
			if c.R != 0 {
				t.Errorf("img0.At(%d, %d): red want %d got %d", i, j, 0, c.R)
			}
		}
	}
}

func TestImageSubImageAt(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Fill(color.RGBA{0xff, 0, 0, 0xff})

	got := img.SubImage(image.Rect(1, 1, 16, 16)).At(0, 0).(color.RGBA)
	want := color.RGBA{}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	got = img.SubImage(image.Rect(1, 1, 16, 16)).At(1, 1).(color.RGBA)
	want = color.RGBA{0xff, 0, 0, 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageSubImageSize(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Fill(color.RGBA{0xff, 0, 0, 0xff})

	got, _ := img.SubImage(image.Rect(1, 1, 16, 16)).(*ebiten.Image).Size()
	want := 15
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageDrawImmediately(t *testing.T) {
	const w, h = 16, 16
	img0 := ebiten.NewImage(w, h)
	img1 := ebiten.NewImage(w, h)
	// Do not manipulate img0 here.

	img0.Fill(color.RGBA{0xff, 0, 0, 0xff})
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img0.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("img0.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}

	img0.DrawImage(img1, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img0.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("img0.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #669, #759
func TestImageLinearFilterGlitch(t *testing.T) {
	const w, h = 200, 12
	const scale = 1.2
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(int(math.Floor(w*scale)), h)

	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := i + w*j
			if j < 3 {
				pix[4*idx] = 0xff
				pix[4*idx+1] = 0xff
				pix[4*idx+2] = 0xff
				pix[4*idx+3] = 0xff
			} else {
				pix[4*idx] = 0
				pix[4*idx+1] = 0
				pix[4*idx+2] = 0
				pix[4*idx+3] = 0xff
			}
		}
	}
	src.WritePixels(pix)

	for _, f := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, 1)
		op.Filter = f
		dst.DrawImage(src, op)

		for j := 1; j < h-1; j++ {
			offset := int(math.Ceil(scale))
			for i := offset; i < int(math.Floor(w*scale))-offset; i++ {
				got := dst.At(i, j).(color.RGBA)
				var want color.RGBA
				if j < 3 {
					want = color.RGBA{0xff, 0xff, 0xff, 0xff}
				} else {
					want = color.RGBA{0, 0, 0, 0xff}
				}
				if got != want {
					t.Errorf("dst.At(%d, %d): filter: %d, got: %v, want: %v", i, j, f, got, want)
				}
			}
		}
	}
}

// Issue #1212
func TestImageLinearFilterGlitch2(t *testing.T) {
	const w, h = 100, 100
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)

	idx := 0
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i+j < 100 {
				pix[4*idx] = 0
				pix[4*idx+1] = 0
				pix[4*idx+2] = 0
				pix[4*idx+3] = 0xff
			} else {
				pix[4*idx] = 0xff
				pix[4*idx+1] = 0xff
				pix[4*idx+2] = 0xff
				pix[4*idx+3] = 0xff
			}
			idx++
		}
	}
	src.WritePixels(pix)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(src, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i+j < 100 {
				want = color.RGBA{0, 0, 0, 0xff}
			} else {
				want = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageAddressRepeat(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			if 4 <= i && i < 8 && 4 <= j && j < 8 {
				pix[idx] = byte(i-4) * 0x10
				pix[idx+1] = byte(j-4) * 0x10
				pix[idx+2] = 0
				pix[idx+3] = 0xff
			} else {
				pix[idx] = 0
				pix[idx+1] = 0
				pix[idx+2] = 0xff
				pix[idx+3] = 0xff
			}
		}
	}
	src.WritePixels(pix)

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
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}
	op := &ebiten.DrawTrianglesOptions{}
	op.Address = ebiten.AddressRepeat
	dst.DrawTriangles(vs, is, src.SubImage(image.Rect(4, 4, 8, 8)).(*ebiten.Image), op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{byte(i%4) * 0x10, byte(j%4) * 0x10, 0, 0xff}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageAddressRepeatNegativePosition(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			if 4 <= i && i < 8 && 4 <= j && j < 8 {
				pix[idx] = byte(i-4) * 0x10
				pix[idx+1] = byte(j-4) * 0x10
				pix[idx+2] = 0
				pix[idx+3] = 0xff
			} else {
				pix[idx] = 0
				pix[idx+1] = 0
				pix[idx+2] = 0xff
				pix[idx+3] = 0xff
			}
		}
	}
	src.WritePixels(pix)

	vs := []ebiten.Vertex{
		{
			DstX:   0,
			DstY:   0,
			SrcX:   -w,
			SrcY:   -h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   w,
			DstY:   0,
			SrcX:   0,
			SrcY:   -h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   0,
			DstY:   h,
			SrcX:   -w,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   w,
			DstY:   h,
			SrcX:   0,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}
	op := &ebiten.DrawTrianglesOptions{}
	op.Address = ebiten.AddressRepeat
	dst.DrawTriangles(vs, is, src.SubImage(image.Rect(4, 4, 8, 8)).(*ebiten.Image), op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{byte(i%4) * 0x10, byte(j%4) * 0x10, 0, 0xff}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageWritePixelsAfterClear(t *testing.T) {
	const w, h = 256, 256
	img := ebiten.NewImage(w, h)
	img.WritePixels(make([]byte, 4*w*h))
	// Clear used to call DrawImage to clear the image, which was the cause of crash. It is because after
	// DrawImage is called, WritePixels for a region is forbidden.
	//
	// Now WritePixels was always called at Clear instead.
	img.Clear()
	img.WritePixels(make([]byte, 4*w*h))

	// The test passes if this doesn't crash.
}

func TestImageSet(t *testing.T) {
	type Pt struct {
		X, Y int
	}

	const w, h = 16, 16
	img := ebiten.NewImage(w, h)
	colors := map[Pt]color.RGBA{
		{1, 2}:   {3, 4, 5, 6},
		{7, 8}:   {9, 10, 11, 12},
		{13, 14}: {15, 16, 17, 18},
		{-1, -1}: {19, 20, 21, 22},
	}

	for p, c := range colors {
		img.Set(p.X, p.Y, c)
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j).(color.RGBA)
			var want color.RGBA
			if c, ok := colors[Pt{i, j}]; ok {
				want = c
			}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageSetAndDraw(t *testing.T) {
	type Pt struct {
		X, Y int
	}

	const w, h = 16, 16
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)
	colors := map[Pt]color.RGBA{
		{1, 2}:   {3, 4, 5, 6},
		{7, 8}:   {9, 10, 11, 12},
		{13, 14}: {15, 16, 17, 18},
	}
	for p, c := range colors {
		src.Set(p.X, p.Y, c)
		dst.Set(p.X+1, p.Y+1, c)
	}

	dst.DrawImage(src, nil)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if c, ok := colors[Pt{i, j}]; ok {
				want = c
			}
			if c, ok := colors[Pt{i - 1, j - 1}]; ok {
				want = c
			}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	src.Clear()
	dst.Clear()
	for p, c := range colors {
		src.Set(p.X, p.Y, c)
		dst.Set(p.X+1, p.Y+1, c)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(2, 2)
	dst.DrawImage(src.SubImage(image.Rect(2, 2, w-2, h-2)).(*ebiten.Image), op)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if 2 <= i && 2 <= j && i < w-2 && j < h-2 {
				if c, ok := colors[Pt{i, j}]; ok {
					want = c
				}
			}
			if c, ok := colors[Pt{i - 1, j - 1}]; ok {
				want = c
			}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageAlphaOnBlack(t *testing.T) {
	const w, h = 16, 16
	src0 := ebiten.NewImage(w, h)
	src1 := ebiten.NewImage(w, h)
	dst0 := ebiten.NewImage(w, h)
	dst1 := ebiten.NewImage(w, h)

	pix0 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if (i/3)%2 == (j/3)%2 {
				pix0[4*(i+j*w)] = 0xff
				pix0[4*(i+j*w)+1] = 0xff
				pix0[4*(i+j*w)+2] = 0xff
				pix0[4*(i+j*w)+3] = 0xff
			}
		}
	}
	src0.WritePixels(pix0)

	pix1 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if (i/3)%2 == (j/3)%2 {
				pix1[4*(i+j*w)] = 0xff
				pix1[4*(i+j*w)+1] = 0xff
				pix1[4*(i+j*w)+2] = 0xff
				pix1[4*(i+j*w)+3] = 0xff
			} else {
				pix1[4*(i+j*w)] = 0
				pix1[4*(i+j*w)+1] = 0
				pix1[4*(i+j*w)+2] = 0
				pix1[4*(i+j*w)+3] = 0xff
			}
		}
	}
	src1.WritePixels(pix1)

	dst0.Fill(color.Black)
	dst1.Fill(color.Black)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.5, 0.5)
	op.Filter = ebiten.FilterLinear
	dst0.DrawImage(src0, op)
	dst1.DrawImage(src1, op)

	gray := false
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst0.At(i, j)
			want := dst1.At(i, j)
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
			if r := got.(color.RGBA).R; 0 < r && r < 255 {
				gray = true
			}
		}
	}
	if !gray {
		t.Errorf("gray must be included in the results but not")
	}
}

func TestImageDrawTrianglesWithSubImage(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)

	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if 4 <= i && i < 8 && 4 <= j && j < 8 {
				pix[4*(i+j*w)] = 0xff
				pix[4*(i+j*w)+1] = 0
				pix[4*(i+j*w)+2] = 0
				pix[4*(i+j*w)+3] = 0xff
			} else {
				pix[4*(i+j*w)] = 0
				pix[4*(i+j*w)+1] = 0xff
				pix[4*(i+j*w)+2] = 0
				pix[4*(i+j*w)+3] = 0xff
			}
		}
	}
	src.WritePixels(pix)

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
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}
	op := &ebiten.DrawTrianglesOptions{}
	op.Address = ebiten.AddressClampToZero
	dst.DrawTriangles(vs, is, src.SubImage(image.Rect(4, 4, 8, 8)).(*ebiten.Image), op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if 4 <= i && i < 8 && 4 <= j && j < 8 {
				want = src.At(i, j).(color.RGBA)
			}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #823
func TestImageAtAfterDisposingSubImage(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Set(0, 0, color.White)
	img.SubImage(image.Rect(0, 0, 16, 16))
	runtime.GC()

	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	want64 := color.RGBA64{0xffff, 0xffff, 0xffff, 0xffff}
	got := img.At(0, 0)
	if got != want {
		t.Errorf("At(0,0) got: %v, want: %v", got, want)
	}
	got = img.RGBA64At(0, 0)
	if got != want64 {
		t.Errorf("RGBA64At(0,0) got: %v, want: %v", got, want)
	}

	img.Set(0, 1, color.White)
	sub := img.SubImage(image.Rect(0, 0, 16, 16)).(*ebiten.Image)
	sub.Dispose()

	got = img.At(0, 1)
	if got != want {
		t.Errorf("At(0,1) got: %v, want: %v", got, want64)
	}
	got = img.RGBA64At(0, 1)
	if got != want64 {
		t.Errorf("RGBA64At(0,1) got: %v, want: %v", got, want64)
	}
}

func TestImageSubImageSubImage(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Fill(color.White)
	sub0 := img.SubImage(image.Rect(0, 0, 12, 12)).(*ebiten.Image)
	sub1 := sub0.SubImage(image.Rect(4, 4, 16, 16)).(*ebiten.Image)
	cases := []struct {
		X     int
		Y     int
		Color color.RGBA
	}{
		{
			X:     0,
			Y:     0,
			Color: color.RGBA{},
		},
		{
			X:     4,
			Y:     4,
			Color: color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			X:     15,
			Y:     15,
			Color: color.RGBA{},
		},
	}
	for _, c := range cases {
		got := sub1.At(c.X, c.Y)
		want := c.Color
		if got != want {
			t.Errorf("At(%d, %d): got: %v, want: %v", c.X, c.Y, got, want)
		}
	}
}

// Issue #839
func TestImageTooSmallMipmap(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)

	src.Fill(color.White)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1, 0.24)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(src.SubImage(image.Rect(5, 0, 6, 16)).(*ebiten.Image), op)
	got := dst.At(0, 0).(color.RGBA)
	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageZeroSizedMipmap(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(src.SubImage(image.ZR).(*ebiten.Image), op)
}

// Issue #898
func TestImageFillingAndEdges(t *testing.T) {
	const (
		srcw, srch = 16, 16
		dstw, dsth = 256, 16
	)

	src := ebiten.NewImage(srcw, srch)
	dst := ebiten.NewImage(dstw, dsth)

	src.Fill(color.White)
	dst.Fill(color.Black)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(dstw-2)/float64(srcw), float64(dsth-2)/float64(srch))
	op.GeoM.Translate(1, 1)
	dst.DrawImage(src, op)

	for j := 0; j < dsth; j++ {
		for i := 0; i < dstw; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if i == 0 || i == dstw-1 || j == 0 || j == dsth-1 {
				want = color.RGBA{0, 0, 0, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageDrawTrianglesAndMutateArgs(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	clr := color.RGBA{0xff, 0, 0, 0xff}
	src.Fill(clr)

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
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}
	dst.DrawTriangles(vs, is, src, nil)
	vs[0].SrcX = w
	vs[0].SrcY = h
	is[5] = 0

	for j := 0; j < w; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := clr
			if got != want {
				t.Errorf("dst.At(%d, %d): got %v, want %v", i, j, got, want)
			}
		}
	}
}

func TestImageWritePixelsOnSubImage(t *testing.T) {
	dst := ebiten.NewImage(17, 31)
	dst.Fill(color.RGBA{0xff, 0, 0, 0xff})

	pix0 := make([]byte, 4*5*3)
	idx := 0
	for j := 0; j < 3; j++ {
		for i := 0; i < 5; i++ {
			pix0[4*idx] = 0
			pix0[4*idx+1] = 0xff
			pix0[4*idx+2] = 0
			pix0[4*idx+3] = 0xff
			idx++
		}
	}
	r0 := image.Rect(4, 5, 9, 8)
	dst.SubImage(r0).(*ebiten.Image).WritePixels(pix0)

	pix1 := make([]byte, 4*5*3)
	idx = 0
	for j := 0; j < 3; j++ {
		for i := 0; i < 5; i++ {
			pix1[4*idx] = 0
			pix1[4*idx+1] = 0
			pix1[4*idx+2] = 0xff
			pix1[4*idx+3] = 0xff
			idx++
		}
	}
	r1 := image.Rect(11, 10, 16, 13)
	dst.SubImage(r1).(*ebiten.Image).WritePixels(pix1)

	// Clear the pixels. This should not affect the result.
	idx = 0
	for j := 0; j < 3; j++ {
		for i := 0; i < 5; i++ {
			pix1[4*idx] = 0
			pix1[4*idx+1] = 0
			pix1[4*idx+2] = 0
			pix1[4*idx+3] = 0
			idx++
		}
	}

	for j := 0; j < 31; j++ {
		for i := 0; i < 17; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0, 0, 0xff}
			p := image.Pt(i, j)
			switch {
			case p.In(r0):
				want = color.RGBA{0, 0xff, 0, 0xff}
			case p.In(r1):
				want = color.RGBA{0, 0, 0xff, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageDrawTrianglesWithColorM(t *testing.T) {
	const w, h = 16, 16
	dst0 := ebiten.NewImage(w, h)
	dst1 := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.White)

	vs0 := []ebiten.Vertex{
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
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	op := &ebiten.DrawTrianglesOptions{}
	op.ColorM.Scale(0.2, 0.4, 0.6, 0.8)
	is := []uint16{0, 1, 2, 1, 2, 3}
	dst0.DrawTriangles(vs0, is, src, op)

	vs1 := []ebiten.Vertex{
		{
			DstX:   0,
			DstY:   0,
			SrcX:   0,
			SrcY:   0,
			ColorR: 0.2,
			ColorG: 0.4,
			ColorB: 0.6,
			ColorA: 0.8,
		},
		{
			DstX:   w,
			DstY:   0,
			SrcX:   w,
			SrcY:   0,
			ColorR: 0.2,
			ColorG: 0.4,
			ColorB: 0.6,
			ColorA: 0.8,
		},
		{
			DstX:   0,
			DstY:   h,
			SrcX:   0,
			SrcY:   h,
			ColorR: 0.2,
			ColorG: 0.4,
			ColorB: 0.6,
			ColorA: 0.8,
		},
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 0.2,
			ColorG: 0.4,
			ColorB: 0.6,
			ColorA: 0.8,
		},
	}
	dst1.DrawTriangles(vs1, is, src, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst0.At(i, j)
			want := dst1.At(i, j)
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageDrawTrianglesInterpolatesColors(t *testing.T) {
	const w, h = 3, 1
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)
	src.Fill(color.White)

	vs := []ebiten.Vertex{
		{
			DstX:   0,
			DstY:   0,
			SrcX:   0,
			SrcY:   0,
			ColorR: 1,
			ColorG: 0,
			ColorB: 0,
			ColorA: 0,
		},
		{
			DstX:   w,
			DstY:   0,
			SrcX:   w,
			SrcY:   0,
			ColorR: 0,
			ColorG: 1,
			ColorB: 0,
			ColorA: 1,
		},
		{
			DstX:   0,
			DstY:   h,
			SrcX:   0,
			SrcY:   h,
			ColorR: 1,
			ColorG: 0,
			ColorB: 0,
			ColorA: 0,
		},
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 0,
			ColorG: 1,
			ColorB: 0,
			ColorA: 1,
		},
	}
	dst.Fill(color.RGBA{0x00, 0x00, 0xff, 0xff})
	op := &ebiten.DrawTrianglesOptions{}
	is := []uint16{0, 1, 2, 1, 2, 3}
	dst.DrawTriangles(vs, is, src, op)

	got := dst.At(1, 0).(color.RGBA)

	// Correct color interpolation uses the alpha channel and notices that colors on the left side of the texture are fully transparent.
	want := color.RGBA{0x00, 0x80, 0x80, 0xff}

	// Interpolation isn't exactly specified, so a range is accepable.
	diff := math.Max(math.Max(math.Max(
		math.Abs(float64(got.R)-float64(want.R)),
		math.Abs(float64(got.G)-float64(want.G))),
		math.Abs(float64(got.B)-float64(want.B))),
		math.Abs(float64(got.A)-float64(want.A)))

	if diff > 5 {
		t.Errorf("At(1, 0): got: %v, want: %v", got, want)
	}
}

func TestImageDrawTrianglesShaderInterpolatesValues(t *testing.T) {
	const w, h = 3, 1
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)
	src.Fill(color.White)

	vs := []ebiten.Vertex{
		{
			DstX:   0,
			DstY:   0,
			SrcX:   0,
			SrcY:   0,
			ColorR: 1,
			ColorG: 0,
			ColorB: 0,
			ColorA: 0,
		},
		{
			DstX:   w,
			DstY:   0,
			SrcX:   w,
			SrcY:   0,
			ColorR: 0,
			ColorG: 1,
			ColorB: 0,
			ColorA: 1,
		},
		{
			DstX:   0,
			DstY:   h,
			SrcX:   0,
			SrcY:   h,
			ColorR: 1,
			ColorG: 0,
			ColorB: 0,
			ColorA: 0,
		},
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 0,
			ColorG: 1,
			ColorB: 0,
			ColorA: 1,
		},
	}
	dst.Fill(color.RGBA{0x00, 0x00, 0xff, 0xff})
	op := &ebiten.DrawTrianglesShaderOptions{
		Images: [4]*ebiten.Image{src, nil, nil, nil},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}
	shader, err := ebiten.NewShader([]byte(`
		package main
		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
			return color
		}
	`))
	if err != nil {
		t.Fatalf("could not compile shader: %v", err)
	}
	dst.DrawTrianglesShader(vs, is, shader, op)

	got := dst.At(1, 0).(color.RGBA)

	// Shaders get each color value interpolated independently.
	want := color.RGBA{0x80, 0x80, 0x80, 0xff}

	// Interpolation isn't exactly specified, so a range is accepable.
	diff := math.Max(math.Max(math.Max(
		math.Abs(float64(got.R)-float64(want.R)),
		math.Abs(float64(got.G)-float64(want.G))),
		math.Abs(float64(got.B)-float64(want.B))),
		math.Abs(float64(got.A)-float64(want.A)))

	if diff > 5 {
		t.Errorf("At(1, 0): got: %v, want: %v", got, want)
	}
}

// Issue #1137
func TestImageDrawOver(t *testing.T) {
	const (
		w = 320
		h = 240
	)
	dst := ebiten.NewImage(w, h)
	src := image.NewUniform(color.RGBA{0xff, 0, 0, 0xff})
	// This must not cause infinite-loop.
	draw.Draw(dst, dst.Bounds(), src, image.ZP, draw.Over)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageDrawDisposedImage(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DrawImage must panic but not")
		}
	}()

	dst := ebiten.NewImage(16, 16)
	src := ebiten.NewImage(16, 16)
	src.Dispose()
	dst.DrawImage(src, nil)
}

func TestImageDrawTrianglesDisposedImage(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DrawTriangles must panic but not")
		}
	}()

	dst := ebiten.NewImage(16, 16)
	src := ebiten.NewImage(16, 16)
	src.Dispose()
	vs := make([]ebiten.Vertex, 4)
	is := []uint16{0, 1, 2, 1, 2, 3}
	dst.DrawTriangles(vs, is, src, nil)
}

// #1137
func BenchmarkImageDrawOver(b *testing.B) {
	dst := ebiten.NewImage(16, 16)
	src := image.NewUniform(color.Black)
	for n := 0; n < b.N; n++ {
		draw.Draw(dst, dst.Bounds(), src, image.ZP, draw.Over)
	}
}

// Issue #1171
func TestImageFloatTranslate(t *testing.T) {
	const w, h = 32, 32

	for s := 2; s <= 8; s++ {
		s := s
		t.Run(fmt.Sprintf("scale%d", s), func(t *testing.T) {
			check := func(src *ebiten.Image) {
				dst := ebiten.NewImage(w*(s+1), h*(s+1))
				dst.Fill(color.RGBA{0xff, 0, 0, 0xff})

				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(s), float64(s))
				op.GeoM.Translate(0, 0.501)
				dst.DrawImage(src, op)

				for j := 0; j < h*s+1; j++ {
					for i := 0; i < w*s; i++ {
						got := dst.At(i, j)
						x := byte(0xff)
						if j > 0 {
							x = (byte(j) - 1) / byte(s)
						}
						want := color.RGBA{x, 0, 0, 0xff}
						if got != want {
							t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
						}
					}
				}
			}

			t.Run("image", func(t *testing.T) {
				src := ebiten.NewImage(w, h)
				pix := make([]byte, 4*w*h)
				for j := 0; j < h; j++ {
					for i := 0; i < w; i++ {
						pix[4*(j*w+i)] = byte(j)
						pix[4*(j*w+i)+3] = 0xff
					}
				}
				src.WritePixels(pix)
				check(src)
			})

			t.Run("subimage", func(t *testing.T) {
				src := ebiten.NewImage(w*s, h*s)
				pix := make([]byte, 4*(w*s)*(h*s))
				for j := 0; j < h*s; j++ {
					for i := 0; i < w*s; i++ {
						pix[4*(j*(w*s)+i)] = byte(j)
						pix[4*(j*(w*s)+i)+3] = 0xff
					}
				}
				src.WritePixels(pix)
				check(src.SubImage(image.Rect(0, 0, w, h)).(*ebiten.Image))
			})
		})
	}
}

// Issue #1213
func TestImageColorMCopy(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	for k := 0; k < 256; k++ {
		op := &ebiten.DrawImageOptions{}
		op.ColorM.Translate(1, 1, 1, float64(k)/0xff)
		op.CompositeMode = ebiten.CompositeModeCopy
		dst.DrawImage(src, op)

		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				got := dst.At(i, j).(color.RGBA)
				want := color.RGBA{byte(k), byte(k), byte(k), byte(k)}
				if !sameColors(got, want, 1) {
					t.Fatalf("dst.At(%d, %d), k: %d: got %v, want %v", i, j, k, got, want)
				}
			}
		}
	}
}

// TODO: Do we have to guarantee this behavior? See #1222
func TestImageWritePixelsAndModifyPixels(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			pix[idx] = 0xff
			pix[idx+1] = 0
			pix[idx+2] = 0
			pix[idx+3] = 0xff
		}
	}

	src.WritePixels(pix)

	// Modify pix after WritePixels
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			pix[idx] = 0
			pix[idx+1] = 0xff
			pix[idx+2] = 0
			pix[idx+3] = 0xff
		}
	}

	// Ensure that src's pixels are actually used
	dst.DrawImage(src, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := src.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("src.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageCompositeModeMultiply(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	dst.Fill(color.RGBA{0x10, 0x20, 0x30, 0x40})
	src.Fill(color.RGBA{0x50, 0x60, 0x70, 0x80})

	op := &ebiten.DrawImageOptions{}
	op.CompositeMode = ebiten.CompositeModeMultiply
	dst.DrawImage(src, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{
				R: byte(math.Floor((0x10 / 255.0) * (0x50 / 255.0) * 255)),
				G: byte(math.Floor((0x20 / 255.0) * (0x60 / 255.0) * 255)),
				B: byte(math.Floor((0x30 / 255.0) * (0x70 / 255.0) * 255)),
				A: byte(math.Floor((0x40 / 255.0) * (0x80 / 255.0) * 255)),
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #1269
func TestImageZeroTriangle(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(1, 1)

	vs := []ebiten.Vertex{}
	is := []uint16{}
	dst.DrawTriangles(vs, is, src, nil)
}

// Issue #1398
func TestImageDrawImageTooBigScale(t *testing.T) {
	dst := ebiten.NewImage(1, 1)
	src := ebiten.NewImage(1, 1)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1e20, 1e20)
	dst.DrawImage(src, op)
}

// Issue #1398
func TestImageDrawImageTooSmallScale(t *testing.T) {
	dst := ebiten.NewImage(1, 1)
	src := ebiten.NewImage(1, 1)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(1e-10, 1e-10)
	dst.DrawImage(src, op)
}

// Issue #1399
func TestImageDrawImageCannotAllocateImageForMipmap(t *testing.T) {
	dst := ebiten.NewImage(1, 1)
	src := ebiten.NewImage(maxImageSize, maxImageSize)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(64, 64)
	dst.DrawImage(src, op)
	dst.At(0, 0)
}

func TestImageNewImageWithZeroSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DrawImage must panic but not")
		}
	}()

	_ = ebiten.NewImage(0, 1)
}

func TestImageNewImageFromImageWithZeroSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DrawImage must panic but not")
		}
	}()

	img := image.NewRGBA(image.Rect(0, 0, 0, 1))
	_ = ebiten.NewImageFromImage(img)
}

func TestImageClip(t *testing.T) {
	const (
		w = 16
		h = 16
	)
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	dst.Fill(color.RGBA{0xff, 0, 0, 0xff})
	src.Fill(color.RGBA{0, 0xff, 0, 0xff})

	dst.SubImage(image.Rect(4, 5, 12, 14)).(*ebiten.Image).DrawImage(src, nil)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if 4 <= i && i < 12 && 5 <= j && j < 14 {
				want = color.RGBA{0, 0xff, 0, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #1691
func TestImageSubImageFill(t *testing.T) {
	dst := ebiten.NewImage(3, 3).SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	dst.Fill(color.White)
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			if i == 1 && j == 1 {
				want = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	dst = ebiten.NewImage(17, 31).SubImage(image.Rect(3, 4, 8, 10)).(*ebiten.Image)
	dst.Fill(color.White)
	for j := 0; j < 31; j++ {
		for i := 0; i < 17; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			if 3 <= i && i < 8 && 4 <= j && j < 10 {
				want = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageEvenOdd(t *testing.T) {
	emptyImage := ebiten.NewImage(3, 3)
	emptyImage.Fill(color.White)
	emptySubImage := emptyImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	vs0 := []ebiten.Vertex{
		{
			DstX: 1, DstY: 1, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 0, ColorB: 0, ColorA: 1,
		},
		{
			DstX: 15, DstY: 1, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 0, ColorB: 0, ColorA: 1,
		},
		{
			DstX: 1, DstY: 15, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 0, ColorB: 0, ColorA: 1,
		},
		{
			DstX: 15, DstY: 15, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 0, ColorB: 0, ColorA: 1,
		},
	}
	is0 := []uint16{0, 1, 2, 1, 2, 3}

	vs1 := []ebiten.Vertex{
		{
			DstX: 2, DstY: 2, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
		},
		{
			DstX: 14, DstY: 2, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
		},
		{
			DstX: 2, DstY: 14, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
		},
		{
			DstX: 14, DstY: 14, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
		},
	}
	is1 := []uint16{4, 5, 6, 5, 6, 7}

	vs2 := []ebiten.Vertex{
		{
			DstX: 3, DstY: 3, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
		},
		{
			DstX: 13, DstY: 3, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
		},
		{
			DstX: 3, DstY: 13, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
		},
		{
			DstX: 13, DstY: 13, SrcX: 1, SrcY: 1,
			ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
		},
	}
	is2 := []uint16{8, 9, 10, 9, 10, 11}

	// Draw all the vertices once. The even-odd rule is applied for all the vertices once.
	dst := ebiten.NewImage(16, 16)
	op := &ebiten.DrawTrianglesOptions{
		FillRule: ebiten.EvenOdd,
	}
	dst.DrawTriangles(append(append(vs0, vs1...), vs2...), append(append(is0, is1...), is2...), emptySubImage, op)
	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			switch {
			case 3 <= i && i < 13 && 3 <= j && j < 13:
				want = color.RGBA{0, 0, 0xff, 0xff}
			case 2 <= i && i < 14 && 2 <= j && j < 14:
				want = color.RGBA{0, 0, 0, 0}
			case 1 <= i && i < 15 && 1 <= j && j < 15:
				want = color.RGBA{0xff, 0, 0, 0xff}
			default:
				want = color.RGBA{0, 0, 0, 0}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Do the same thing but with a little shift. This confirms that the underlying stencil buffer is cleared correctly.
	for i := range vs0 {
		vs0[i].DstX++
		vs0[i].DstY++
	}
	for i := range vs1 {
		vs1[i].DstX++
		vs1[i].DstY++
	}
	for i := range vs2 {
		vs2[i].DstX++
		vs2[i].DstY++
	}
	dst.Clear()
	dst.DrawTriangles(append(append(vs0, vs1...), vs2...), append(append(is0, is1...), is2...), emptySubImage, op)
	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			switch {
			case 4 <= i && i < 14 && 4 <= j && j < 14:
				want = color.RGBA{0, 0, 0xff, 0xff}
			case 3 <= i && i < 15 && 3 <= j && j < 15:
				want = color.RGBA{0, 0, 0, 0}
			case 2 <= i && i < 16 && 2 <= j && j < 16:
				want = color.RGBA{0xff, 0, 0, 0xff}
			default:
				want = color.RGBA{0, 0, 0, 0}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Do the same thing but with split DrawTriangle calls. This confirms that the even-odd rule is applied for one call.
	for i := range vs0 {
		vs0[i].DstX--
		vs0[i].DstY--
	}
	for i := range vs1 {
		vs1[i].DstX--
		vs1[i].DstY--
	}
	for i := range vs2 {
		vs2[i].DstX--
		vs2[i].DstY--
	}
	dst.Clear()
	// Use the first indices set.
	dst.DrawTriangles(vs0, is0, emptySubImage, op)
	dst.DrawTriangles(vs1, is0, emptySubImage, op)
	dst.DrawTriangles(vs2, is0, emptySubImage, op)
	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			switch {
			case 3 <= i && i < 13 && 3 <= j && j < 13:
				want = color.RGBA{0, 0, 0xff, 0xff}
			case 2 <= i && i < 14 && 2 <= j && j < 14:
				want = color.RGBA{0, 0xff, 0, 0xff}
			case 1 <= i && i < 15 && 1 <= j && j < 15:
				want = color.RGBA{0xff, 0, 0, 0xff}
			default:
				want = color.RGBA{0, 0, 0, 0}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// #1658
func BenchmarkColorMScale(b *testing.B) {
	r := rand.Float64
	dst := ebiten.NewImage(16, 16)
	src := ebiten.NewImage(16, 16)
	for n := 0; n < b.N; n++ {
		op := &ebiten.DrawImageOptions{}
		op.ColorM.Scale(r(), r(), r(), r())
		dst.DrawImage(src, op)
	}
}

func TestIndicesOverflow(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.White)

	op := &ebiten.DrawTrianglesOptions{}
	vs := make([]ebiten.Vertex, 3)
	is := make([]uint16, graphics.IndicesCount/3*3)
	dst.DrawTriangles(vs, is, src, op)

	// Cause an overflow for indices.
	vs = []ebiten.Vertex{
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
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is = []uint16{0, 1, 2, 1, 2, 3}
	dst.DrawTriangles(vs, is, src, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestVerticesOverflow(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.White)

	op := &ebiten.DrawTrianglesOptions{}
	vs := make([]ebiten.Vertex, graphics.IndicesCount-1)
	is := make([]uint16, 3)
	dst.DrawTriangles(vs, is, src, op)

	// Cause an overflow for vertices.
	vs = []ebiten.Vertex{
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
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is = []uint16{0, 1, 2, 1, 2, 3}
	dst.DrawTriangles(vs, is, src, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestTooManyVertices(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.White)

	op := &ebiten.DrawTrianglesOptions{}
	vs := make([]ebiten.Vertex, graphics.IndicesCount+1)
	is := make([]uint16, 3)
	dst.DrawTriangles(vs, is, src, op)

	// Force to cause flushing the graphics commands.
	// Confirm this doesn't freeze.
	dst.At(0, 0)
}

func TestImageNewImageFromEbitenImage(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			pix[idx] = byte(i)
			pix[idx+1] = byte(j)
			pix[idx+2] = 0
			pix[idx+3] = 0xff
		}
	}

	img0 := ebiten.NewImage(w, h)
	img0.WritePixels(pix)
	img1 := ebiten.NewImageFromImage(img0)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img1.At(i, j)
			want := color.RGBA{byte(i), byte(j), 0, 0xff}
			if got != want {
				t.Errorf("img1.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	img2 := ebiten.NewImageFromImage(img0.SubImage(image.Rect(4, 4, 12, 12)))
	for j := 0; j < h/2; j++ {
		for i := 0; i < w/2; i++ {
			got := img2.At(i, j)
			want := color.RGBA{byte(i + 4), byte(j + 4), 0, 0xff}
			if got != want {
				t.Errorf("img1.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageOptionsUnmanaged(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			pix[idx] = byte(i)
			pix[idx+1] = byte(j)
			pix[idx+2] = 0
			pix[idx+3] = 0xff
		}
	}

	op := &ebiten.NewImageOptions{
		Unmanaged: true,
	}
	img := ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), op)
	img.WritePixels(pix)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want := color.RGBA{byte(i), byte(j), 0, 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageOptionsNegativeBoundsWritePixels(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	pix0 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			pix0[idx] = byte(i)
			pix0[idx+1] = byte(j)
			pix0[idx+2] = 0
			pix0[idx+3] = 0xff
		}
	}

	const offset = -8
	img := ebiten.NewImageWithOptions(image.Rect(offset, offset, w+offset, h+offset), nil)
	img.WritePixels(pix0)

	for j := offset; j < h+offset; j++ {
		for i := offset; i < w+offset; i++ {
			got := img.At(i, j)
			want := color.RGBA{byte(i - offset), byte(j - offset), 0, 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	pix1 := make([]byte, 4*(w/2)*(h/2))
	for j := 0; j < h/2; j++ {
		for i := 0; i < w/2; i++ {
			idx := 4 * (i + j*w/2)
			pix1[idx] = 0
			pix1[idx+1] = 0
			pix1[idx+2] = 0xff
			pix1[idx+3] = 0xff
		}
	}

	const offset2 = -4
	sub := image.Rect(offset2, offset2, w/2+offset2, h/2+offset2)
	img.SubImage(sub).(*ebiten.Image).WritePixels(pix1)
	for j := offset; j < h+offset; j++ {
		for i := offset; i < w+offset; i++ {
			got := img.At(i, j)
			want := color.RGBA{byte(i - offset), byte(j - offset), 0, 0xff}
			if image.Pt(i, j).In(sub) {
				want = color.RGBA{0, 0, 0xff, 0xff}
			}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageOptionsNegativeBoundsSet(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	pix0 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + j*w)
			pix0[idx] = byte(i)
			pix0[idx+1] = byte(j)
			pix0[idx+2] = 0
			pix0[idx+3] = 0xff
		}
	}

	const offset = -8
	img := ebiten.NewImageWithOptions(image.Rect(offset, offset, w+offset, h+offset), nil)
	img.WritePixels(pix0)
	img.Set(-1, -2, color.RGBA{0, 0, 0, 0})

	for j := offset; j < h+offset; j++ {
		for i := offset; i < w+offset; i++ {
			got := img.At(i, j)
			want := color.RGBA{byte(i - offset), byte(j - offset), 0, 0xff}
			if i == -1 && j == -2 {
				want = color.RGBA{0, 0, 0, 0}
			}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageOptionsNegativeBoundsDrawImage(t *testing.T) {
	const (
		w      = 16
		h      = 16
		offset = -8
	)
	dst := ebiten.NewImageWithOptions(image.Rect(offset, offset, w+offset, h+offset), nil)
	src := ebiten.NewImageWithOptions(image.Rect(-1, -1, 1, 1), nil)
	pix := make([]byte, 4*2*2)
	for i := range pix {
		pix[i] = 0xff
	}
	src.WritePixels(pix)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-1, -1)
	op.GeoM.Scale(2, 3)
	dst.DrawImage(src, op)
	for j := offset; j < h+offset; j++ {
		for i := offset; i < w+offset; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			if -2 <= i && i < 2 && -3 <= j && j < 3 {
				want = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageOptionsNegativeBoundsDrawTriangles(t *testing.T) {
	const (
		w      = 16
		h      = 16
		offset = -8
	)
	dst := ebiten.NewImageWithOptions(image.Rect(offset, offset, w+offset, h+offset), nil)
	src := ebiten.NewImageWithOptions(image.Rect(-1, -1, 1, 1), nil)
	pix := make([]byte, 4*2*2)
	for i := range pix {
		pix[i] = 0xff
	}
	src.WritePixels(pix)
	vs := []ebiten.Vertex{
		{
			DstX:   -2,
			DstY:   -3,
			SrcX:   -1,
			SrcY:   -1,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   2,
			DstY:   -3,
			SrcX:   1,
			SrcY:   -1,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   -2,
			DstY:   3,
			SrcX:   -1,
			SrcY:   1,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   2,
			DstY:   3,
			SrcX:   1,
			SrcY:   1,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}
	dst.DrawTriangles(vs, is, src, nil)
	for j := offset; j < h+offset; j++ {
		for i := offset; i < w+offset; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			if -2 <= i && i < 2 && -3 <= j && j < 3 {
				want = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageFromImageOptions(t *testing.T) {
	r := image.Rect(-2, -3, 4, 5)
	pix := make([]byte, 4*r.Dx()*r.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	src := &image.RGBA{
		Pix:    pix,
		Stride: 4 * 2,
		Rect:   r,
	}

	op := &ebiten.NewImageFromImageOptions{
		PreserveBounds: true,
	}
	img := ebiten.NewImageFromImageWithOptions(src, op)
	if got, want := img.Bounds(), r; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	for j := r.Min.Y; j < r.Max.Y; j++ {
		for i := r.Min.X; i < r.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageFromEbitenImageOptions(t *testing.T) {
	r := image.Rect(-2, -3, 4, 5)
	src := ebiten.NewImageWithOptions(r, nil)
	pix := make([]byte, 4*r.Dx()*r.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	src.WritePixels(pix)

	op := &ebiten.NewImageFromImageOptions{
		PreserveBounds: true,
	}
	img := ebiten.NewImageFromImageWithOptions(src, op)
	if got, want := img.Bounds(), r; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	for j := r.Min.Y; j < r.Max.Y; j++ {
		for i := r.Min.X; i < r.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2159
func TestImageOptionsFill(t *testing.T) {
	r0 := image.Rect(-2, -3, 4, 5)
	img := ebiten.NewImageWithOptions(r0, nil)
	img.Fill(color.RGBA{0xff, 0, 0, 0xff})
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	r1 := image.Rect(-1, -2, 3, 4)
	img.SubImage(r1).(*ebiten.Image).Fill(color.RGBA{0, 0xff, 0, 0xff})
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if image.Pt(i, j).In(r1) {
				want = color.RGBA{0, 0xff, 0, 0xff}
			}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2159
func TestImageOptionsClear(t *testing.T) {
	r0 := image.Rect(-2, -3, 4, 5)
	img := ebiten.NewImageWithOptions(r0, nil)
	img.Fill(color.RGBA{0xff, 0, 0, 0xff})
	img.Clear()
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{0, 0, 0, 0}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	img.Fill(color.RGBA{0xff, 0, 0, 0xff})
	r1 := image.Rect(-1, -2, 3, 4)
	img.SubImage(r1).(*ebiten.Image).Clear()
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{0xff, 0, 0, 0xff}
			if image.Pt(i, j).In(r1) {
				want = color.RGBA{0, 0, 0, 0}
			}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2178
func TestImageTooManyDrawImage(t *testing.T) {
	src := ebiten.NewImage(1, 1)
	src.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})

	const (
		w = 256
		h = 256
	)
	dst := ebiten.NewImage(w, h)

	op := &ebiten.DrawImageOptions{}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(i), float64(j))
			dst.DrawImage(src, op)
		}
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if got, want := dst.At(i, j), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2178
func TestImageTooManyDrawTriangles(t *testing.T) {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	src := img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	const (
		w = 128
		h = 64
	)
	dst := ebiten.NewImage(w, h)

	var vertices []ebiten.Vertex
	var indices []uint16
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			n := uint16(len(vertices))
			vertices = append(vertices,
				ebiten.Vertex{
					DstX:   float32(i),
					DstY:   float32(j),
					SrcX:   1,
					SrcY:   1,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
				ebiten.Vertex{
					DstX:   float32(i) + 1,
					DstY:   float32(j),
					SrcX:   2,
					SrcY:   1,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
				ebiten.Vertex{
					DstX:   float32(i),
					DstY:   float32(j) + 1,
					SrcX:   1,
					SrcY:   2,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
				ebiten.Vertex{
					DstX:   float32(i) + 1,
					DstY:   float32(j) + 1,
					SrcX:   2,
					SrcY:   2,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
			)
			indices = append(indices, n, n+1, n+2, n+1, n+2, n+3)
		}
	}
	dst.DrawTriangles(vertices, indices, src, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if got, want := dst.At(i, j), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageSetOverSet(t *testing.T) {
	img := ebiten.NewImage(1, 1)
	img.Set(0, 0, color.RGBA{0xff, 0xff, 0xff, 0xff})
	if got, want := img.At(0, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Apply the change by 'Set' by calling DrawImage.
	dummy := ebiten.NewImage(1, 1)
	img.DrawImage(dummy, nil)
	if got, want := img.At(0, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	img.Set(0, 0, color.RGBA{0x80, 0x80, 0x80, 0x80})
	if got, want := img.At(0, 0), (color.RGBA{0x80, 0x80, 0x80, 0x80}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Apply the change by 'Set' again.
	img.DrawImage(dummy, nil)
	if got, want := img.At(0, 0), (color.RGBA{0x80, 0x80, 0x80, 0x80}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #2204
func TestImageTooManyConstantBuffersInDirectX(t *testing.T) {
	src := ebiten.NewImage(3, 3)
	src.Fill(color.White)
	src = src.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	vs := []ebiten.Vertex{
		{
			DstX: 0, DstY: 0, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		},
		{
			DstX: 16, DstY: 0, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		},
		{
			DstX: 0, DstY: 16, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		},
		{
			DstX: 16, DstY: 16, SrcX: 1, SrcY: 1,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}

	dst0 := ebiten.NewImage(16, 16)
	dst1 := ebiten.NewImage(16, 16)
	op := &ebiten.DrawTrianglesOptions{
		FillRule: ebiten.EvenOdd,
	}
	for i := 0; i < 100; i++ {
		dst0.DrawTriangles(vs, is, src, op)
		dst1.DrawTriangles(vs, is, src, op)
	}

	if got, want := dst0.At(0, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst1.At(0, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
