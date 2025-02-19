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
	"math/rand/v2"
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// maxImageSize is a maximum image size that should work in almost every environment.
const maxImageSize = 4096 - 2

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

	w, h := img0.Bounds().Dx(), img0.Bounds().Dy()
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
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			want := color.RGBAModel.Convert(img.At(i, j))
			if got != want {
				t.Errorf("(%d, %d): got %v; want %v", i, j, got, want)
			}
		}
	}
}

func TestImageComposition(t *testing.T) {
	img2Color := color.NRGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88}
	img3Color := color.NRGBA{R: 0x85, G: 0xa3, B: 0x08, A: 0xd3}

	// TODO: Rename this to img0
	img1, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}

	w, h := img1.Bounds().Dx(), img1.Bounds().Dy()

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
	// Note that mutex usages: without defer, unlocking is not called when panicking.
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
		w, h := img0.Bounds().Dx(), img0.Bounds().Dy()
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
	w, h := img0.Bounds().Dx(), img0.Bounds().Dy()
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
	w, h := img0.Bounds().Dx(), img0.Bounds().Dy()
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
	// Create a dummy image so that the shared texture is used and origImg's position is shifted.
	dummyImg := ebiten.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 16, 16)))
	defer dummyImg.Deallocate()

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
	for j := 0; j < img0.Bounds().Dy(); j++ {
		for i := 0; i < img0.Bounds().Dx(); i++ {
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
	for j := 0; j < img0.Bounds().Dy(); j++ {
		for i := 0; i < img0.Bounds().Dx(); i++ {
			got := img0.At(i, j)
			want := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}
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

func TestImageDeallocate(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Fill(color.White)
	img.Deallocate()

	// The color is transparent (color.RGBA{}).
	got := img.At(0, 0)
	want := color.RGBA{}
	if got != want {
		t.Errorf("img.At(0, 0) got: %v, want: %v", got, want)
	}
}

func TestImageBlendLighter(t *testing.T) {
	img0, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}

	w, h := img0.Bounds().Dx(), img0.Bounds().Dy()
	img1 := ebiten.NewImage(w, h)
	img1.Fill(color.RGBA{R: 0x01, G: 0x02, B: 0x03, A: 0x04})
	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendLighter
	img1.DrawImage(img0, op)
	for j := 0; j < img1.Bounds().Dy(); j++ {
		for i := 0; i < img1.Bounds().Dx(); i++ {
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
	w2, h2 := eimg.Bounds().Dx(), eimg.Bounds().Dy()
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
			want := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}
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
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
	red := color.RGBA{R: 0xff, A: 0xff}
	transparent := color.RGBA{}

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
					w, h := img0.Bounds().Dx(), img0.Bounds().Dy()
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
						is := []uint16{0, 1, 2, 1, 2, 3}
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
		src.Fill(color.RGBA{R: c, G: c, B: c, A: 0xff})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(i), 0)
		dst.DrawImage(src, op)
	}

	for i := 0; i < width; i++ {
		c := indexToColor(i)
		got := dst.At(i, 0).(color.RGBA)
		want := color.RGBA{R: c, G: c, B: c, A: 0xff}
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

func BenchmarkDrawTriangles(b *testing.B) {
	const w, h = 16, 16
	img0 := ebiten.NewImage(w, h)
	img1 := ebiten.NewImage(w, h)
	op := &ebiten.DrawTrianglesOptions{}
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
	for i := 0; i < b.N; i++ {
		img0.DrawTriangles(vs, is, img1, op)
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
	src.Fill(color.RGBA{R: 0xff, A: 0xff})

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
				want := color.RGBA{}
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
	src.Fill(color.RGBA{R: 0xff, A: 0xff})

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
	gotW, gotH := img.Bounds().Dx(), img.Bounds().Dy()
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
	want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
		want := color.RGBA{R: uint8(i + j), G: uint8((i + j) >> 8), B: uint8((i + j) >> 16), A: 0xff}
		if got != want {
			t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
		}
	}
	for j := 4095; j < 4096; j++ {
		i := 4095
		got := dst.At(i, j).(color.RGBA)
		want := color.RGBA{R: uint8(i + j), G: uint8((i + j) >> 8), B: uint8((i + j) >> 16), A: 0xff}
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

		dh := dst.Bounds().Dy()
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
					want = color.RGBA{R: 0xff, A: 0xff}
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
	src.Fill(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
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
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
	w, h := src.Bounds().Dx(), src.Bounds().Dy()

	l1 := ebiten.NewImage(w/2, h/2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = ebiten.FilterLinear
	l1.DrawImage(src, op)

	l1w, l1h := l1.Bounds().Dx(), l1.Bounds().Dy()
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
	w, h := src.Bounds().Dx(), src.Bounds().Dy()

	l1 := ebiten.NewImage(w/2, h/2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = ebiten.FilterLinear
	l1.DrawImage(src, op)

	l1w, l1h := l1.Bounds().Dx(), l1.Bounds().Dy()
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
		op.ColorScale.Scale(1, 1, 0, 1)
		img0.DrawImage(img1, op)

		op.GeoM.Translate(128, 0)
		op.ColorScale.Reset()
		op.ColorScale.Scale(0, 1, 1, 1)
		img0.DrawImage(img1, op)

		want := color.RGBA{G: 0xff, B: 0xff, A: 0xff}
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
	img1.Fill(color.RGBA{R: 0xff, A: 0xff})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.25, 0.25)
	op.Filter = ebiten.FilterLinear
	img0.DrawImage(img1, op)

	// Call DrawTriangle on img1 and fill it with green
	img2.Fill(color.RGBA{G: 0xff, A: 0xff})
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

	w, h := img0.Bounds().Dx(), img0.Bounds().Dy()
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
	img.Fill(color.RGBA{R: 0xff, A: 0xff})

	got := img.SubImage(image.Rect(1, 1, 16, 16)).At(0, 0).(color.RGBA)
	want := color.RGBA{}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	got = img.SubImage(image.Rect(1, 1, 16, 16)).At(1, 1).(color.RGBA)
	want = color.RGBA{R: 0xff, A: 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageSubImageSize(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Fill(color.RGBA{R: 0xff, A: 0xff})

	got := img.SubImage(image.Rect(1, 1, 16, 16)).Bounds().Dx()
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

	img0.Fill(color.RGBA{R: 0xff, A: 0xff})
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img0.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, A: 0xff}
			if got != want {
				t.Errorf("img0.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}

	img0.DrawImage(img1, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img0.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, A: 0xff}
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
					want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
				} else {
					want = color.RGBA{A: 0xff}
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
				want = color.RGBA{A: 0xff}
			} else {
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
			want := color.RGBA{R: byte(i%4) * 0x10, G: byte(j%4) * 0x10, A: 0xff}
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
			want := color.RGBA{R: byte(i%4) * 0x10, G: byte(j%4) * 0x10, A: 0xff}
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

	want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	want64 := color.RGBA64{R: 0xffff, G: 0xffff, B: 0xffff, A: 0xffff}
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

func TestImageAtAfterDeallocateSubImage(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	img.Set(0, 0, color.White)
	img.SubImage(image.Rect(0, 0, 16, 16))
	runtime.GC()

	want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	want64 := color.RGBA64{R: 0xffff, G: 0xffff, B: 0xffff, A: 0xffff}
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
	sub.Deallocate()

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
			Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
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
	want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if i == 0 || i == dstw-1 || j == 0 || j == dsth-1 {
				want = color.RGBA{A: 0xff}
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
	clr := color.RGBA{R: 0xff, A: 0xff}
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
	dst.Fill(color.RGBA{R: 0xff, A: 0xff})

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
			want := color.RGBA{R: 0xff, A: 0xff}
			p := image.Pt(i, j)
			switch {
			case p.In(r0):
				want = color.RGBA{G: 0xff, A: 0xff}
			case p.In(r1):
				want = color.RGBA{B: 0xff, A: 0xff}
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

	for _, format := range []ebiten.ColorScaleMode{
		ebiten.ColorScaleModeStraightAlpha,
		ebiten.ColorScaleModePremultipliedAlpha,
	} {
		format := format
		t.Run(fmt.Sprintf("format%d", format), func(t *testing.T) {
			var cr, cg, cb, ca float32
			switch format {
			case ebiten.ColorScaleModeStraightAlpha:
				// The values are the same as ColorM.Scale
				cr = 0.2
				cg = 0.4
				cb = 0.6
				ca = 0.8
			case ebiten.ColorScaleModePremultipliedAlpha:
				cr = 0.2 * 0.8
				cg = 0.4 * 0.8
				cb = 0.6 * 0.8
				ca = 0.8
			}
			vs1 := []ebiten.Vertex{
				{
					DstX:   0,
					DstY:   0,
					SrcX:   0,
					SrcY:   0,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
				{
					DstX:   w,
					DstY:   0,
					SrcX:   w,
					SrcY:   0,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
				{
					DstX:   0,
					DstY:   h,
					SrcX:   0,
					SrcY:   h,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
				{
					DstX:   w,
					DstY:   h,
					SrcX:   w,
					SrcY:   h,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
			}

			dst1 := ebiten.NewImage(w, h)
			op := &ebiten.DrawTrianglesOptions{}
			op.ColorScaleMode = format
			dst1.DrawTriangles(vs1, is, src, op)

			for j := 0; j < h; j++ {
				for i := 0; i < w; i++ {
					got := dst0.At(i, j)
					want := dst1.At(i, j)
					if got != want {
						t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}
		})
	}
}

func TestImageDrawTrianglesInterpolatesColors(t *testing.T) {
	const w, h = 3, 1
	src := ebiten.NewImage(w, h)
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

	for _, format := range []ebiten.ColorScaleMode{
		ebiten.ColorScaleModeStraightAlpha,
		ebiten.ColorScaleModePremultipliedAlpha,
	} {
		format := format
		t.Run(fmt.Sprintf("format%d", format), func(t *testing.T) {
			dst := ebiten.NewImage(w, h)
			dst.Fill(color.RGBA{B: 0xff, A: 0xff})

			op := &ebiten.DrawTrianglesOptions{}
			op.ColorScaleMode = format

			is := []uint16{0, 1, 2, 1, 2, 3}
			dst.DrawTriangles(vs, is, src, op)

			got := dst.At(1, 0).(color.RGBA)

			// Correct color interpolation uses the alpha channel
			// and notices that colors on the left side of the texture are fully transparent.
			var want color.RGBA
			switch format {
			case ebiten.ColorScaleModeStraightAlpha:
				want = color.RGBA{G: 0x80, B: 0x80, A: 0xff}
			case ebiten.ColorScaleModePremultipliedAlpha:
				want = color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
			}

			if !sameColors(got, want, 2) {
				t.Errorf("At(1, 0): got: %v, want: %v", got, want)
			}
		})
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
	dst.Fill(color.RGBA{B: 0xff, A: 0xff})
	op := &ebiten.DrawTrianglesShaderOptions{
		Images: [4]*ebiten.Image{src, nil, nil, nil},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}
	shader, err := ebiten.NewShader([]byte(`
		package main
		func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
			return color
		}
	`))
	if err != nil {
		t.Fatalf("could not compile shader: %v", err)
	}
	dst.DrawTrianglesShader(vs, is, shader, op)

	got := dst.At(1, 0).(color.RGBA)

	// Shaders get each color value interpolated independently.
	want := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}

	if !sameColors(got, want, 2) {
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
	src := image.NewUniform(color.RGBA{R: 0xff, A: 0xff})
	// This must not cause infinite-loop.
	draw.Draw(dst, dst.Bounds(), src, image.ZP, draw.Over)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := color.RGBA{R: 0xff, A: 0xff}
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

func TestImageDrawDeallocatedImage(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	src := ebiten.NewImage(16, 16)
	src.Deallocate()
	// DrawImage must not panic.
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

func TestImageDrawTrianglesDeallocateImage(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	src := ebiten.NewImage(16, 16)
	src.Deallocate()
	vs := make([]ebiten.Vertex, 4)
	is := []uint16{0, 1, 2, 1, 2, 3}
	// DrawTriangles must not panic.
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
				dst.Fill(color.RGBA{R: 0xff, A: 0xff})

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
						want := color.RGBA{R: x, A: 0xff}
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
		op.Blend = ebiten.BlendCopy
		dst.DrawImage(src, op)

		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				got := dst.At(i, j).(color.RGBA)
				want := color.RGBA{R: byte(k), G: byte(k), B: byte(k), A: byte(k)}
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
			want := color.RGBA{R: 0xff, A: 0xff}
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

	dst.Fill(color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0x40})
	src.Fill(color.RGBA{R: 0x50, G: 0x60, B: 0x70, A: 0x80})

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

	dst.Fill(color.RGBA{R: 0xff, A: 0xff})
	src.Fill(color.RGBA{G: 0xff, A: 0xff})

	dst.SubImage(image.Rect(4, 5, 12, 14)).(*ebiten.Image).DrawImage(src, nil)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, A: 0xff}
			if 4 <= i && i < 12 && 5 <= j && j < 14 {
				want = color.RGBA{G: 0xff, A: 0xff}
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
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageEvenOdd(t *testing.T) {
	whiteImage := ebiten.NewImage(3, 3)
	whiteImage.Fill(color.White)
	emptySubImage := whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

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
		FillRule: ebiten.FillRuleEvenOdd,
	}
	dst.DrawTriangles(append(append(vs0, vs1...), vs2...), append(append(is0, is1...), is2...), emptySubImage, op)
	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			switch {
			case 3 <= i && i < 13 && 3 <= j && j < 13:
				want = color.RGBA{B: 0xff, A: 0xff}
			case 2 <= i && i < 14 && 2 <= j && j < 14:
				want = color.RGBA{}
			case 1 <= i && i < 15 && 1 <= j && j < 15:
				want = color.RGBA{R: 0xff, A: 0xff}
			default:
				want = color.RGBA{}
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
				want = color.RGBA{B: 0xff, A: 0xff}
			case 3 <= i && i < 15 && 3 <= j && j < 15:
				want = color.RGBA{}
			case 2 <= i && i < 16 && 2 <= j && j < 16:
				want = color.RGBA{R: 0xff, A: 0xff}
			default:
				want = color.RGBA{}
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
				want = color.RGBA{B: 0xff, A: 0xff}
			case 2 <= i && i < 14 && 2 <= j && j < 14:
				want = color.RGBA{G: 0xff, A: 0xff}
			case 1 <= i && i < 15 && 1 <= j && j < 15:
				want = color.RGBA{R: 0xff, A: 0xff}
			default:
				want = color.RGBA{}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageFillRule(t *testing.T) {
	for _, fillRule := range []ebiten.FillRule{ebiten.FillRuleFillAll, ebiten.FillRuleNonZero, ebiten.FillRuleEvenOdd} {
		fillRule := fillRule
		var name string
		switch fillRule {
		case ebiten.FillRuleFillAll:
			name = "FillAll"
		case ebiten.FillRuleNonZero:
			name = "NonZero"
		case ebiten.FillRuleEvenOdd:
			name = "EvenOdd"
		}
		t.Run(name, func(t *testing.T) {
			whiteImage := ebiten.NewImage(3, 3)
			whiteImage.Fill(color.White)
			emptySubImage := whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

			// The outside rectangle (clockwise)
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
					DstX: 15, DstY: 15, SrcX: 1, SrcY: 1,
					ColorR: 1, ColorG: 0, ColorB: 0, ColorA: 1,
				},
				{
					DstX: 1, DstY: 15, SrcX: 1, SrcY: 1,
					ColorR: 1, ColorG: 0, ColorB: 0, ColorA: 1,
				},
			}
			is0 := []uint16{0, 1, 2, 2, 3, 0}

			// An inside rectangle (clockwise)
			vs1 := []ebiten.Vertex{
				{
					DstX: 2, DstY: 2, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
				},
				{
					DstX: 7, DstY: 2, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
				},
				{
					DstX: 7, DstY: 7, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
				},
				{
					DstX: 2, DstY: 7, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 1, ColorB: 0, ColorA: 1,
				},
			}
			is1 := []uint16{4, 5, 6, 6, 7, 4}

			// An inside rectangle (counter-clockwise)
			vs2 := []ebiten.Vertex{
				{
					DstX: 9, DstY: 9, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
				},
				{
					DstX: 14, DstY: 9, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
				},
				{
					DstX: 14, DstY: 14, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
				},
				{
					DstX: 9, DstY: 14, SrcX: 1, SrcY: 1,
					ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
				},
			}
			is2 := []uint16{8, 11, 10, 10, 9, 8}

			// Draw all the vertices once. The even-odd rule is applied for all the vertices once.
			dst := ebiten.NewImage(16, 16)
			op := &ebiten.DrawTrianglesOptions{
				FillRule: fillRule,
			}
			dst.DrawTriangles(append(append(vs0, vs1...), vs2...), append(append(is0, is1...), is2...), emptySubImage, op)
			for j := 0; j < 16; j++ {
				for i := 0; i < 16; i++ {
					got := dst.At(i, j)
					var want color.RGBA
					switch {
					case 2 <= i && i < 7 && 2 <= j && j < 7:
						if fillRule != ebiten.FillRuleEvenOdd {
							want = color.RGBA{G: 0xff, A: 0xff}
						}
					case 9 <= i && i < 14 && 9 <= j && j < 14:
						if fillRule == ebiten.FillRuleFillAll {
							want = color.RGBA{B: 0xff, A: 0xff}
						}
					case 1 <= i && i < 15 && 1 <= j && j < 15:
						want = color.RGBA{R: 0xff, A: 0xff}
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
					case 3 <= i && i < 8 && 3 <= j && j < 8:
						if fillRule != ebiten.FillRuleEvenOdd {
							want = color.RGBA{G: 0xff, A: 0xff}
						}
					case 10 <= i && i < 15 && 10 <= j && j < 15:
						if fillRule == ebiten.FillRuleFillAll {
							want = color.RGBA{B: 0xff, A: 0xff}
						}
					case 2 <= i && i < 16 && 2 <= j && j < 16:
						want = color.RGBA{R: 0xff, A: 0xff}
					}
					if got != want {
						t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}

			// Do the same thing but with split DrawTriangle calls. This confirms that fill rules are applied for one call.
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
			dst.DrawTriangles(vs0, []uint16{0, 1, 2, 2, 3, 0}, emptySubImage, op)
			dst.DrawTriangles(vs1, []uint16{0, 1, 2, 2, 3, 0}, emptySubImage, op)
			dst.DrawTriangles(vs2, []uint16{0, 3, 2, 2, 1, 0}, emptySubImage, op)
			for j := 0; j < 16; j++ {
				for i := 0; i < 16; i++ {
					got := dst.At(i, j)
					var want color.RGBA
					switch {
					case 2 <= i && i < 7 && 2 <= j && j < 7:
						want = color.RGBA{G: 0xff, A: 0xff}
					case 9 <= i && i < 14 && 9 <= j && j < 14:
						want = color.RGBA{B: 0xff, A: 0xff}
					case 1 <= i && i < 15 && 1 <= j && j < 15:
						want = color.RGBA{R: 0xff, A: 0xff}
					}
					if got != want {
						t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}
		})
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

func TestImageMoreIndicesThanMaxUint16(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.White)

	op := &ebiten.DrawTrianglesOptions{}
	vs := make([]ebiten.Vertex, 3)
	is := make([]uint16, 65538)
	dst.DrawTriangles(vs, is, src, op)

	// The next draw call should work well (and this is likely batched).
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
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageMoreVerticesThanMaxUint16(t *testing.T) {
	const (
		w = 16
		h = 16
	)

	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.White)

	op := &ebiten.DrawTrianglesOptions{}
	vs := make([]ebiten.Vertex, math.MaxUint16+1)
	is := make([]uint16, 3)
	dst.DrawTriangles(vs, is, src, op)

	// The next draw call should work well (and this is likely batched).
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
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
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
			want := color.RGBA{R: byte(i), G: byte(j), A: 0xff}
			if got != want {
				t.Errorf("img1.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	img2 := ebiten.NewImageFromImage(img0.SubImage(image.Rect(4, 4, 12, 12)))
	for j := 0; j < h/2; j++ {
		for i := 0; i < w/2; i++ {
			got := img2.At(i, j)
			want := color.RGBA{R: byte(i + 4), G: byte(j + 4), A: 0xff}
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
			want := color.RGBA{R: byte(i), G: byte(j), A: 0xff}
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
			want := color.RGBA{R: byte(i - offset), G: byte(j - offset), A: 0xff}
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
			want := color.RGBA{R: byte(i - offset), G: byte(j - offset), A: 0xff}
			if image.Pt(i, j).In(sub) {
				want = color.RGBA{B: 0xff, A: 0xff}
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
	img.Set(-1, -2, color.RGBA{})

	for j := offset; j < h+offset; j++ {
		for i := offset; i < w+offset; i++ {
			got := img.At(i, j)
			want := color.RGBA{R: byte(i - offset), G: byte(j - offset), A: 0xff}
			if i == -1 && j == -2 {
				want = color.RGBA{}
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
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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
	img.Fill(color.RGBA{R: 0xff, A: 0xff})
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{R: 0xff, A: 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	r1 := image.Rect(-1, -2, 3, 4)
	img.SubImage(r1).(*ebiten.Image).Fill(color.RGBA{G: 0xff, A: 0xff})
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{R: 0xff, A: 0xff}
			if image.Pt(i, j).In(r1) {
				want = color.RGBA{G: 0xff, A: 0xff}
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
	img.Fill(color.RGBA{R: 0xff, A: 0xff})
	img.Clear()
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	img.Fill(color.RGBA{R: 0xff, A: 0xff})
	r1 := image.Rect(-1, -2, 3, 4)
	img.SubImage(r1).(*ebiten.Image).Clear()
	for j := r0.Min.Y; j < r0.Max.Y; j++ {
		for i := r0.Min.X; i < r0.Max.X; i++ {
			got := img.At(i, j)
			want := color.RGBA{R: 0xff, A: 0xff}
			if image.Pt(i, j).In(r1) {
				want = color.RGBA{}
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
	src.Fill(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

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
			if got, want := dst.At(i, j), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageTooManyDrawImage2(t *testing.T) {
	src := ebiten.NewImage(1, 1)
	src.Fill(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

	const (
		w = 512
		h = 512
	)
	dst := ebiten.NewImage(w, h)

	posToColor := func(i, j int) color.RGBA {
		return color.RGBA{
			R: byte(i),
			G: byte(j),
			B: 0xff,
			A: 0xff,
		}
	}

	op := &ebiten.DrawImageOptions{}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(i), float64(j))
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(posToColor(i, j))
			dst.DrawImage(src, op)
		}
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if got, want := dst.At(i, j).(color.RGBA), posToColor(i, j); !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2178
func TestImageTooManyDrawTriangles(t *testing.T) {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
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
			if got, want := dst.At(i, j), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageSetOverSet(t *testing.T) {
	img := ebiten.NewImage(1, 1)
	img.Set(0, 0, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	if got, want := img.At(0, 0), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Apply the change by 'Set' by calling DrawImage.
	dummy := ebiten.NewImage(1, 1)
	img.DrawImage(dummy, nil)
	if got, want := img.At(0, 0), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	img.Set(0, 0, color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80})
	if got, want := img.At(0, 0), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Apply the change by 'Set' again.
	img.DrawImage(dummy, nil)
	if got, want := img.At(0, 0), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}); got != want {
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
		FillRule: ebiten.FillRuleEvenOdd,
	}
	for i := 0; i < 100; i++ {
		dst0.DrawTriangles(vs, is, src, op)
		dst1.DrawTriangles(vs, is, src, op)
	}

	if got, want := dst0.At(0, 0), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst1.At(0, 0), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageColorMAndScale(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)

	src.Fill(color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80})
	vs := []ebiten.Vertex{
		{
			SrcX:   0,
			SrcY:   0,
			DstX:   0,
			DstY:   0,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
		{
			SrcX:   w,
			SrcY:   0,
			DstX:   w,
			DstY:   0,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
		{
			SrcX:   0,
			SrcY:   h,
			DstX:   0,
			DstY:   h,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
		{
			SrcX:   w,
			SrcY:   h,
			DstX:   w,
			DstY:   h,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}

	for _, format := range []ebiten.ColorScaleMode{
		ebiten.ColorScaleModeStraightAlpha,
		ebiten.ColorScaleModePremultipliedAlpha,
	} {
		format := format
		t.Run(fmt.Sprintf("format%d", format), func(t *testing.T) {
			dst := ebiten.NewImage(w, h)

			op := &ebiten.DrawTrianglesOptions{}
			op.ColorM.Translate(0.25, 0.25, 0.25, 0)
			op.ColorScaleMode = format
			dst.DrawTriangles(vs, is, src, op)

			got := dst.At(0, 0).(color.RGBA)
			alphaBeforeScale := 0.5
			var want color.RGBA
			switch format {
			case ebiten.ColorScaleModeStraightAlpha:
				want = color.RGBA{
					R: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5 * 0.75)),
					G: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.25 * 0.75)),
					B: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5 * 0.75)),
					A: byte(math.Floor(0xff * alphaBeforeScale * 0.75)),
				}
			case ebiten.ColorScaleModePremultipliedAlpha:
				want = color.RGBA{
					R: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5)),
					G: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.25)),
					B: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5)),
					A: byte(math.Floor(0xff * alphaBeforeScale * 0.75)),
				}
			}
			if !sameColors(got, want, 2) {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

func TestImageBlendOperation(t *testing.T) {
	const w, h = 16, 1
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	dstColor := func(i int) (byte, byte, byte, byte) {
		return byte(4 * i * 17), byte(4*i*17 + 1), byte(4*i*17 + 2), byte(4*i*17 + 3)
	}
	srcColor := func(i int) (byte, byte, byte, byte) {
		return byte(4 * i * 13), byte(4*i*13 + 1), byte(4*i*13 + 2), byte(4*i*13 + 3)
	}
	clamp := func(x int) byte {
		if x > 255 {
			return 255
		}
		if x < 0 {
			return 0
		}
		return byte(x)
	}

	dstPix := make([]byte, 4*w*h)
	for i := 0; i < w; i++ {
		r, g, b, a := dstColor(i)
		dstPix[4*i] = r
		dstPix[4*i+1] = g
		dstPix[4*i+2] = b
		dstPix[4*i+3] = a
	}
	srcPix := make([]byte, 4*w*h)
	for i := 0; i < w; i++ {
		r, g, b, a := srcColor(i)
		srcPix[4*i] = r
		srcPix[4*i+1] = g
		srcPix[4*i+2] = b
		srcPix[4*i+3] = a
	}
	src.WritePixels(srcPix)

	operations := []ebiten.BlendOperation{
		ebiten.BlendOperationAdd,
		ebiten.BlendOperationSubtract,
		ebiten.BlendOperationReverseSubtract,
	}
	for _, rgbOp := range operations {
		for _, alphaOp := range operations {
			// Reset the destination state.
			dst.WritePixels(dstPix)
			op := &ebiten.DrawImageOptions{}
			op.Blend = ebiten.Blend{
				BlendFactorSourceRGB:        ebiten.BlendFactorOne,
				BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
				BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
				BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
				BlendOperationRGB:           rgbOp,
				BlendOperationAlpha:         alphaOp,
			}
			dst.DrawImage(src, op)
			for i := 0; i < w; i++ {
				got := dst.At(i, 0).(color.RGBA)

				sr, sg, sb, sa := srcColor(i)
				dr, dg, db, da := dstColor(i)

				var want color.RGBA
				switch rgbOp {
				case ebiten.BlendOperationAdd:
					want.R = clamp(int(sr) + int(dr))
					want.G = clamp(int(sg) + int(dg))
					want.B = clamp(int(sb) + int(db))
				case ebiten.BlendOperationSubtract:
					want.R = clamp(int(sr) - int(dr))
					want.G = clamp(int(sg) - int(dg))
					want.B = clamp(int(sb) - int(db))
				case ebiten.BlendOperationReverseSubtract:
					want.R = clamp(int(dr) - int(sr))
					want.G = clamp(int(dg) - int(sg))
					want.B = clamp(int(db) - int(sb))
				}
				switch alphaOp {
				case ebiten.BlendOperationAdd:
					want.A = clamp(int(sa) + int(da))
				case ebiten.BlendOperationSubtract:
					want.A = clamp(int(sa) - int(da))
				case ebiten.BlendOperationReverseSubtract:
					want.A = clamp(int(da) - int(sa))
				}

				if !sameColors(got, want, 1) {
					t.Errorf("dst.At(%d, 0): operations: %d, %d: got: %v, want: %v", i, rgbOp, alphaOp, got, want)
				}
			}
		}
	}
}

func TestImageBlendOperationMinAndMax(t *testing.T) {
	const w, h = 16, 1
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	dstColor := func(i int) (byte, byte, byte, byte) {
		return byte(4 * i * 17), byte(4*i*17 + 1), byte(4*i*17 + 2), byte(4*i*17 + 3)
	}
	srcColor := func(i int) (byte, byte, byte, byte) {
		return byte(4 * i * 13), byte(4*i*13 + 1), byte(4*i*13 + 2), byte(4*i*13 + 3)
	}

	dstPix := make([]byte, 4*w*h)
	for i := 0; i < w; i++ {
		r, g, b, a := dstColor(i)
		dstPix[4*i] = r
		dstPix[4*i+1] = g
		dstPix[4*i+2] = b
		dstPix[4*i+3] = a
	}
	srcPix := make([]byte, 4*w*h)
	for i := 0; i < w; i++ {
		r, g, b, a := srcColor(i)
		srcPix[4*i] = r
		srcPix[4*i+1] = g
		srcPix[4*i+2] = b
		srcPix[4*i+3] = a
	}
	src.WritePixels(srcPix)

	operations := []ebiten.BlendOperation{
		ebiten.BlendOperationMin,
		ebiten.BlendOperationMax,
	}
	for _, rgbOp := range operations {
		for _, alphaOp := range operations {
			// Reset the destination state.
			dst.WritePixels(dstPix)
			op := &ebiten.DrawImageOptions{}
			// Use the default blend factors, and confirm that the factors are ignored.
			op.Blend = ebiten.Blend{
				BlendFactorSourceRGB:        ebiten.BlendFactorDefault,
				BlendFactorSourceAlpha:      ebiten.BlendFactorDefault,
				BlendFactorDestinationRGB:   ebiten.BlendFactorDefault,
				BlendFactorDestinationAlpha: ebiten.BlendFactorDefault,
				BlendOperationRGB:           rgbOp,
				BlendOperationAlpha:         alphaOp,
			}
			dst.DrawImage(src, op)
			for i := 0; i < w; i++ {
				got := dst.At(i, 0).(color.RGBA)

				sr, sg, sb, sa := srcColor(i)
				dr, dg, db, da := dstColor(i)

				var want color.RGBA
				switch rgbOp {
				case ebiten.BlendOperationMin:
					want.R = min(sr, dr)
					want.G = min(sg, dg)
					want.B = min(sb, db)
				case ebiten.BlendOperationMax:
					want.R = max(sr, dr)
					want.G = max(sg, dg)
					want.B = max(sb, db)
				}
				switch alphaOp {
				case ebiten.BlendOperationMin:
					want.A = min(sa, da)
				case ebiten.BlendOperationMax:
					want.A = max(sa, da)
				}

				if !sameColors(got, want, 1) {
					t.Errorf("dst.At(%d, 0): operations: %d, %d: got: %v, want: %v", i, rgbOp, alphaOp, got, want)
				}
			}
		}
	}
}

func TestImageBlendFactor(t *testing.T) {
	if skipTooSlowTests(t) {
		return
	}

	const w, h = 16, 1
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	dstColor := func(i int) (byte, byte, byte, byte) {
		return byte(4 * i * 17), byte(4*i*17 + 1), byte(4*i*17 + 2), byte(4*i*17 + 3)
	}
	srcColor := func(i int) (byte, byte, byte, byte) {
		return byte(4 * i * 13), byte(4*i*13 + 1), byte(4*i*13 + 2), byte(4*i*13 + 3)
	}
	colorToFloats := func(r, g, b, a byte) (float64, float64, float64, float64) {
		return float64(r) / 0xff, float64(g) / 0xff, float64(b) / 0xff, float64(a) / 0xff
	}
	clamp := func(x int) byte {
		if x > 255 {
			return 255
		}
		if x < 0 {
			return 0
		}
		return byte(x)
	}

	dstPix := make([]byte, 4*w*h)
	for i := 0; i < w; i++ {
		r, g, b, a := dstColor(i)
		dstPix[4*i] = r
		dstPix[4*i+1] = g
		dstPix[4*i+2] = b
		dstPix[4*i+3] = a
	}
	srcPix := make([]byte, 4*w*h)
	for i := 0; i < w; i++ {
		r, g, b, a := srcColor(i)
		srcPix[4*i] = r
		srcPix[4*i+1] = g
		srcPix[4*i+2] = b
		srcPix[4*i+3] = a
	}
	src.WritePixels(srcPix)

	factors := []ebiten.BlendFactor{
		ebiten.BlendFactorZero,
		ebiten.BlendFactorOne,
		ebiten.BlendFactorSourceColor,
		ebiten.BlendFactorOneMinusSourceColor,
		ebiten.BlendFactorSourceAlpha,
		ebiten.BlendFactorOneMinusSourceAlpha,
		ebiten.BlendFactorDestinationColor,
		ebiten.BlendFactorOneMinusDestinationColor,
		ebiten.BlendFactorDestinationAlpha,
		ebiten.BlendFactorOneMinusDestinationAlpha,
	}
	for _, srcRGBFactor := range factors {
		for _, srcAlphaFactor := range factors {
			for _, dstRGBFactor := range factors {
				for _, dstAlphaFactor := range factors {
					// Reset the destination state.
					dst.WritePixels(dstPix)
					op := &ebiten.DrawImageOptions{}
					op.Blend = ebiten.Blend{
						BlendFactorSourceRGB:        srcRGBFactor,
						BlendFactorSourceAlpha:      srcAlphaFactor,
						BlendFactorDestinationRGB:   dstRGBFactor,
						BlendFactorDestinationAlpha: dstAlphaFactor,
						BlendOperationRGB:           ebiten.BlendOperationAdd,
						BlendOperationAlpha:         ebiten.BlendOperationAdd,
					}
					dst.DrawImage(src, op)
					for i := 0; i < w; i++ {
						got := dst.At(i, 0).(color.RGBA)

						sr, sg, sb, sa := colorToFloats(srcColor(i))
						dr, dg, db, da := colorToFloats(dstColor(i))

						var r, g, b, a float64

						switch srcRGBFactor {
						case ebiten.BlendFactorZero:
							r += 0 * sr
							g += 0 * sg
							b += 0 * sb
						case ebiten.BlendFactorOne:
							r += 1 * sr
							g += 1 * sg
							b += 1 * sb
						case ebiten.BlendFactorSourceColor:
							r += sr * sr
							g += sg * sg
							b += sb * sb
						case ebiten.BlendFactorOneMinusSourceColor:
							r += (1 - sr) * sr
							g += (1 - sg) * sg
							b += (1 - sb) * sb
						case ebiten.BlendFactorSourceAlpha:
							r += sa * sr
							g += sa * sg
							b += sa * sb
						case ebiten.BlendFactorOneMinusSourceAlpha:
							r += (1 - sa) * sr
							g += (1 - sa) * sg
							b += (1 - sa) * sb
						case ebiten.BlendFactorDestinationColor:
							r += dr * sr
							g += dg * sg
							b += db * sb
						case ebiten.BlendFactorOneMinusDestinationColor:
							r += (1 - dr) * sr
							g += (1 - dg) * sg
							b += (1 - db) * sb
						case ebiten.BlendFactorDestinationAlpha:
							r += da * sr
							g += da * sg
							b += da * sb
						case ebiten.BlendFactorOneMinusDestinationAlpha:
							r += (1 - da) * sr
							g += (1 - da) * sg
							b += (1 - da) * sb
						}
						switch srcAlphaFactor {
						case ebiten.BlendFactorZero:
							a += 0 * sa
						case ebiten.BlendFactorOne:
							a += 1 * sa
						case ebiten.BlendFactorSourceColor, ebiten.BlendFactorSourceAlpha:
							a += sa * sa
						case ebiten.BlendFactorOneMinusSourceColor, ebiten.BlendFactorOneMinusSourceAlpha:
							a += (1 - sa) * sa
						case ebiten.BlendFactorDestinationColor, ebiten.BlendFactorDestinationAlpha:
							a += da * sa
						case ebiten.BlendFactorOneMinusDestinationColor, ebiten.BlendFactorOneMinusDestinationAlpha:
							a += (1 - da) * sa
						}

						switch dstRGBFactor {
						case ebiten.BlendFactorZero:
							r += 0 * dr
							g += 0 * dg
							b += 0 * db
						case ebiten.BlendFactorOne:
							r += 1 * dr
							g += 1 * dg
							b += 1 * db
						case ebiten.BlendFactorSourceColor:
							r += sr * dr
							g += sg * dg
							b += sb * db
						case ebiten.BlendFactorOneMinusSourceColor:
							r += (1 - sr) * dr
							g += (1 - sg) * dg
							b += (1 - sb) * db
						case ebiten.BlendFactorSourceAlpha:
							r += sa * dr
							g += sa * dg
							b += sa * db
						case ebiten.BlendFactorOneMinusSourceAlpha:
							r += (1 - sa) * dr
							g += (1 - sa) * dg
							b += (1 - sa) * db
						case ebiten.BlendFactorDestinationColor:
							r += dr * dr
							g += dg * dg
							b += db * db
						case ebiten.BlendFactorOneMinusDestinationColor:
							r += (1 - dr) * dr
							g += (1 - dg) * dg
							b += (1 - db) * db
						case ebiten.BlendFactorDestinationAlpha:
							r += da * dr
							g += da * dg
							b += da * db
						case ebiten.BlendFactorOneMinusDestinationAlpha:
							r += (1 - da) * dr
							g += (1 - da) * dg
							b += (1 - da) * db
						}
						switch dstAlphaFactor {
						case ebiten.BlendFactorZero:
							a += 0 * da
						case ebiten.BlendFactorOne:
							a += 1 * da
						case ebiten.BlendFactorSourceColor, ebiten.BlendFactorSourceAlpha:
							a += sa * da
						case ebiten.BlendFactorOneMinusSourceColor, ebiten.BlendFactorOneMinusSourceAlpha:
							a += (1 - sa) * da
						case ebiten.BlendFactorDestinationColor, ebiten.BlendFactorDestinationAlpha:
							a += da * da
						case ebiten.BlendFactorOneMinusDestinationColor, ebiten.BlendFactorOneMinusDestinationAlpha:
							a += (1 - da) * da
						}

						want := color.RGBA{
							R: clamp(int(r * 0xff)),
							G: clamp(int(g * 0xff)),
							B: clamp(int(b * 0xff)),
							A: clamp(int(a * 0xff)),
						}
						if !sameColors(got, want, 1) {
							t.Errorf("dst.At(%d, 0): factors: %d, %d, %d, %d: got: %v, want: %v", i, srcRGBFactor, srcAlphaFactor, dstRGBFactor, dstAlphaFactor, got, want)
						}
					}
				}
			}
		}
	}
}

func TestImageAntiAlias(t *testing.T) {
	// This value depends on internal/ui.bigOffscreenScale. Sync this.
	const bigOffscreenScale = 2

	const w, h = 272, 208

	dst0 := ebiten.NewImage(w, h)
	dst1 := ebiten.NewImage(w, h)
	tmp := ebiten.NewImage(w*bigOffscreenScale, h*bigOffscreenScale)
	src := ebiten.NewImage(3, 3)
	src.Fill(color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88})

	for _, blend := range []ebiten.Blend{
		{}, // Default
		ebiten.BlendClear,
		ebiten.BlendCopy,
		ebiten.BlendSourceOver,
		ebiten.BlendDestinationOver,
		ebiten.BlendXor,
		ebiten.BlendLighter,
	} {
		rnd := rand.New(rand.NewPCG(0, 0))
		max := func(x, y, z byte) byte {
			if x >= y && x >= z {
				return x
			}
			if y >= x && y >= z {
				return y
			}
			return z
		}

		dstPix := make([]byte, 4*w*h)
		for i := 0; i < w*h; i++ {
			n := rnd.Int()
			r, g, b := byte(n), byte(n>>8), byte(n>>16)
			a := max(r, g, b)
			dstPix[4*i] = r
			dstPix[4*i+1] = g
			dstPix[4*i+2] = b
			dstPix[4*i+3] = a
		}
		dst0.WritePixels(dstPix)
		dst1.WritePixels(dstPix)

		tmp.Clear()

		// Create an actual result.
		op := &ebiten.DrawTrianglesOptions{}
		op.Blend = blend
		op.AntiAlias = true
		vs0 := []ebiten.Vertex{
			{
				DstX:   w / 4,
				DstY:   h / 4,
				SrcX:   1,
				SrcY:   1,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   2 * w / 4,
				DstY:   h / 4,
				SrcX:   2,
				SrcY:   1,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   w / 4,
				DstY:   2 * h / 4,
				SrcX:   1,
				SrcY:   2,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
		}
		is := []uint16{0, 1, 2}
		dst0.DrawTriangles(vs0, is, src, op)

		vs1 := []ebiten.Vertex{
			{
				DstX:   2 * w / 4,
				DstY:   3 * h / 4,
				SrcX:   1,
				SrcY:   2,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   3 * w / 4,
				DstY:   2 * h / 4,
				SrcX:   2,
				SrcY:   1,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   3 * w / 4,
				DstY:   3 * h / 4,
				SrcX:   2,
				SrcY:   2,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
		}
		dst0.DrawTriangles(vs1, is, src, op)

		// Create an expected result.
		// Copy an enlarged destination image to the offscreen.
		opCopy := &ebiten.DrawImageOptions{}
		opCopy.GeoM.Scale(bigOffscreenScale, bigOffscreenScale)
		opCopy.Blend = ebiten.BlendCopy
		tmp.DrawImage(dst1, opCopy)

		// Render the vertices onto the offscreen.
		for i := range vs0 {
			vs0[i].DstX *= 2
			vs0[i].DstY *= 2
		}
		for i := range vs1 {
			vs1[i].DstX *= 2
			vs1[i].DstY *= 2
		}
		op = &ebiten.DrawTrianglesOptions{}
		op.Blend = blend
		tmp.DrawTriangles(vs0, is, src, op)
		tmp.DrawTriangles(vs1, is, src, op)

		// Render a shrunk offscreen image onto the destination.
		opShrink := &ebiten.DrawImageOptions{}
		opShrink.GeoM.Scale(1.0/bigOffscreenScale, 1.0/bigOffscreenScale)
		opShrink.Filter = ebiten.FilterLinear
		opShrink.Blend = ebiten.BlendCopy
		dst1.DrawImage(tmp, opShrink)

		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				got := dst0.At(i, j).(color.RGBA)
				want := dst1.At(i, j).(color.RGBA)
				if !sameColors(got, want, 2) {
					t.Errorf("At(%d, %d), blend: %v, got: %v, want: %v", i, j, blend, got, want)
				}
			}
		}
	}
}

func TestImageColorMScale(t *testing.T) {
	const w, h = 16, 16
	dst0 := ebiten.NewImage(w, h)
	dst1 := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88})

	// As the ColorM is a diagonal matrix, a built-in shader for a color matrix is NOT used.
	op := &ebiten.DrawImageOptions{}
	op.ColorM.Scale(0.3, 0.4, 0.5, 0.6)
	dst0.DrawImage(src, op)

	// As the ColorM is not a diagonal matrix, a built-in shader for a color matrix is used.
	op = &ebiten.DrawImageOptions{}
	op.ColorM.Scale(0.3, 0.4, 0.5, 0.6)
	op.ColorM.Translate(0, 0, 0, 1e-4)
	dst1.DrawImage(src, op)

	got := dst0.At(0, 0)
	want := dst1.At(0, 0)
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageColorScaleAndColorM(t *testing.T) {
	const w, h = 16, 16
	dst0 := ebiten.NewImage(w, h)
	dst1 := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88})

	// ColorScale is applied to premultiplied-alpha colors.
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(0.3*0.6, 0.4*0.6, 0.5*0.6, 0.6)
	dst0.DrawImage(src, op)

	// ColorM.Scale is applied to straight-alpha colors.
	op = &ebiten.DrawImageOptions{}
	op.ColorM.Scale(0.3, 0.4, 0.5, 0.6)
	dst1.DrawImage(src, op)

	got := dst0.At(0, 0)
	want := dst1.At(0, 0)
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #2428
func TestImageSetAndSubImage(t *testing.T) {
	const w, h = 16, 16
	img := ebiten.NewImage(w, h)
	img.Set(1, 1, color.RGBA{R: 0xff, A: 0xff})
	got := img.SubImage(image.Rect(0, 0, w, h)).At(1, 1).(color.RGBA)
	want := color.RGBA{R: 0xff, A: 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #2611
func TestImageDrawTrianglesWithGreaterIndexThanVerticesCount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DrawTriangles must panic but not")
		}
	}()

	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	vs := make([]ebiten.Vertex, 4)
	is := []uint16{0, 1, 2, 1, 2, 4}
	dst.DrawTriangles(vs, is, src, nil)
}

// Issue #2611
func TestImageDrawTrianglesShaderWithGreaterIndexThanVerticesCount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DrawTrianglesShader must panic but not")
		}
	}()

	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)

	vs := make([]ebiten.Vertex, 4)
	is := []uint16{0, 1, 2, 1, 2, 4}
	shader, err := ebiten.NewShader([]byte(`
		package main
		func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
			return color
		}
	`))
	if err != nil {
		t.Fatalf("could not compile shader: %v", err)
	}
	dst.DrawTrianglesShader(vs, is, shader, nil)
}

// Issue #2733
func TestImageGeoMAfterDraw(t *testing.T) {
	src := ebiten.NewImage(1, 1)
	dst := ebiten.NewImageWithOptions(image.Rect(-1, -1, 0, 0), nil)
	op0 := &ebiten.DrawImageOptions{}
	dst.DrawImage(src, op0)
	if x, y := op0.GeoM.Apply(0, 0); x != 0 || y != 0 {
		t.Errorf("got: (%0.2f, %0.2f), want: (0, 0)", x, y)
	}

	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1)
}
`))
	if err != nil {
		t.Fatal(err)
	}
	op1 := &ebiten.DrawRectShaderOptions{}
	dst.DrawRectShader(1, 1, s, op1)
	if x, y := op1.GeoM.Apply(0, 0); x != 0 || y != 0 {
		t.Errorf("got: (%0.2f, %0.2f), want: (0, 0)", x, y)
	}
}

func TestImageWritePixelAndDispose(t *testing.T) {
	const (
		w = 16
		h = 16
	)
	img := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	img.WritePixels(pix)
	img.Dispose()

	// Confirm that any pixel information is invalidated after Dispose is called.
	if got, want := img.At(0, 0), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageWritePixelAndDeallocate(t *testing.T) {
	const (
		w = 16
		h = 16
	)
	img := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	img.WritePixels(pix)
	img.Deallocate()

	// Confirm that any pixel information is cleared after Deallocate is called.
	if got, want := img.At(0, 0), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageDrawImageAfterDeallocation(t *testing.T) {
	src, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}

	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := ebiten.NewImage(w, h)

	dst.DrawImage(src, nil)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := src.At(i, j)
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Even after deallocating the image, the image is still available.
	dst.Deallocate()

	dst.DrawImage(src, nil)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := src.At(i, j)
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2798
func TestImageInvalidPremultipliedAlphaColor(t *testing.T) {
	// This test checks the rendering result when Set and WritePixels use an invalid premultiplied alpha color.
	// The result values are kept and not clamped.

	const (
		w = 16
		h = 16
	)

	dst := ebiten.NewImage(w, h)
	dst.Set(0, 0, color.RGBA{R: 0xff, G: 0xc0, B: 0x80, A: 0x40})
	dst.Set(0, 1, color.RGBA{R: 0xff, G: 0xc0, B: 0x80, A: 0x00})
	if got, want := dst.At(0, 0).(color.RGBA), (color.RGBA{R: 0xff, G: 0xc0, B: 0x80, A: 0x40}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst.At(0, 1).(color.RGBA), (color.RGBA{R: 0xff, G: 0xc0, B: 0x80, A: 0x00}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			pix[4*(j*16+i)] = byte(i)
			pix[4*(j*16+i)+1] = byte(j)
			pix[4*(j*16+i)+2] = 0x80
			pix[4*(j*16+i)+3] = byte(i - j)
		}
	}
	dst.WritePixels(pix)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j)
			want := color.RGBA{R: byte(i), G: byte(j), B: 0x80, A: byte(i - j)}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageDrawTriangles32(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (j*w + i)
			pix[idx] = byte(i) * 0x10
			pix[idx+1] = byte(j) * 0x10
			pix[idx+2] = 0xff
			pix[idx+3] = 0xff
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
	is := []uint32{0, 1, 2, 1, 2, 3}
	op := &ebiten.DrawTrianglesOptions{}
	dst.DrawTriangles32(vs, is, src, op)
	// Even if the indices are modified, this should not affect the rendering result.
	for i := range is {
		is[i] = 0
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: byte(i) * 0x10, G: byte(j) * 0x10, B: 0xff, A: 0xff}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestImageDrawTrianglesShader32(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)

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
	is := []uint32{0, 1, 2, 1, 2, 3}
	op := &ebiten.DrawTrianglesShaderOptions{}
	shader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// imageSrcNOrigin is actually not necessary here.
	// As no source image is bounded, the source's origin position is (0, 0).
	// However, let's use this function for readability.
	p := srcPos - imageSrc0Origin()
	return vec4(floor(p.x) / 16, floor(p.y) / 16, 1, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}
	dst.DrawTrianglesShader32(vs, is, shader, op)
	// Even if the indices are modified, this should not affect the rendering result.
	for i := range is {
		is[i] = 0
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: byte(i) * 0x10, G: byte(j) * 0x10, B: 0xff, A: 0xff}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}
