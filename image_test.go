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
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math"
	"os"
	"runtime"
	"testing"

	. "github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/testflock"
)

func TestMain(m *testing.M) {
	testflock.Lock()
	defer testflock.Unlock()

	code := 0
	// Run an Ebiten process so that (*Image).At is available.
	regularTermination := errors.New("regular termination")
	f := func(screen *Image) error {
		code = m.Run()
		return regularTermination
	}
	if err := Run(f, 320, 240, 1, "Test"); err != nil && err != regularTermination {
		panic(err)
	}
	os.Exit(code)
}

func openEbitenImage() (*Image, image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		return nil, nil, err
	}

	eimg, err := NewImageFromImage(img, FilterNearest)
	if err != nil {
		return nil, nil, err
	}
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
				t.Errorf("img0 At(%d, %d): got %#v; want %#v", i, j, got, want)
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

	img2, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}

	img3, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}

	if err := img2.Fill(img2Color); err != nil {
		t.Fatal(err)
		return
	}
	if err := img3.Fill(img3Color); err != nil {
		t.Fatal(err)
		return
	}
	img_12_3, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := img2.DrawImage(img1, nil); err != nil {
		t.Fatal(err)
		return
	}
	if err := img3.DrawImage(img2, nil); err != nil {
		t.Fatal(err)
		return
	}
	if err := img_12_3.DrawImage(img3, nil); err != nil {
		t.Fatal(err)
		return
	}

	if err := img2.Fill(img2Color); err != nil {
		t.Fatal(err)
		return
	}
	if err := img3.Fill(img3Color); err != nil {
		t.Fatal(err)
		return
	}
	img_1_23, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := img3.DrawImage(img2, nil); err != nil {
		t.Fatal(err)
		return
	}
	if err := img3.DrawImage(img1, nil); err != nil {
		t.Fatal(err)
		return
	}
	if err := img_1_23.DrawImage(img3, nil); err != nil {
		t.Fatal(err)
		return
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c1 := img_12_3.At(i, j).(color.RGBA)
			c2 := img_1_23.At(i, j).(color.RGBA)
			if !sameColors(c1, c2, 1) {
				t.Errorf("img_12_3.At(%d, %d) = %#v; img_1_23.At(%[1]d, %[2]d) = %#[4]v", i, j, c1, c2)
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
		img1, err := NewImage(w*scale, h*scale, FilterNearest)
		if err != nil {
			t.Fatal(err)
			return
		}
		op := &DrawImageOptions{}
		op.GeoM.Scale(float64(scale), float64(scale))

		if err := img1.DrawImage(img0, op); err != nil {
			t.Fatal(err)
			return
		}

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
	img1, err := NewImage(h, w, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	op := &DrawImageOptions{}
	op.GeoM.Rotate(math.Pi / 2)
	op.GeoM.Translate(float64(h), 0)
	if err := img1.DrawImage(img0, op); err != nil {
		t.Fatal(err)
		return
	}

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
	img1, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	op := &DrawImageOptions{}
	op.GeoM.Rotate(math.Pi)
	op.GeoM.Translate(float64(w), float64(h))
	if err := img1.DrawImage(img0, op); err != nil {
		t.Fatal(err)
		return
	}

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

func TestImageReplacePixels(t *testing.T) {
	// Create a dummy image so that the shared texture is used and origImg's position is shfited.
	dummyImg, _ := NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 16, 16)), FilterDefault)
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
	img0, err := NewImage(size.X, size.Y, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}

	if err := img0.ReplacePixels(img.Pix); err != nil {
		t.Fatal(err)
		return
	}
	for j := 0; j < img0.Bounds().Size().Y; j++ {
		for i := 0; i < img0.Bounds().Size().X; i++ {
			got := img0.At(i, j)
			want := img.At(i, j)
			if got != want {
				t.Errorf("img0 At(%d, %d): got %#v; want %#v", i, j, got, want)
			}
		}
	}

	p := make([]uint8, 4*size.X*size.Y)
	for i := range p {
		p[i] = 0x80
	}
	if err := img0.ReplacePixels(p); err != nil {
		t.Fatal(err)
		return
	}
	// Even if p is changed after calling ReplacePixel, img0 uses the original values.
	for i := range p {
		p[i] = 0
	}
	for j := 0; j < img0.Bounds().Size().Y; j++ {
		for i := 0; i < img0.Bounds().Size().X; i++ {
			got := img0.At(i, j)
			want := color.RGBA{0x80, 0x80, 0x80, 0x80}
			if got != want {
				t.Errorf("img0 At(%d, %d): got %#v; want %#v", i, j, got, want)
			}
		}
	}
}

func TestImageReplacePixelsNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("ReplacePixels(nil) must panic")
		}
	}()

	img, err := NewImage(16, 16, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	img.Fill(color.White)
	img.ReplacePixels(nil)
}

func TestImageDispose(t *testing.T) {
	img, err := NewImage(16, 16, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	img.Fill(color.White)
	if err := img.Dispose(); err != nil {
		t.Errorf("img.Dipose() returns error: %v", err)
	}

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
	img1, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := img1.Fill(color.RGBA{0x01, 0x02, 0x03, 0x04}); err != nil {
		t.Fatal(err)
		return
	}
	op := &DrawImageOptions{}
	op.CompositeMode = CompositeModeLighter
	if err := img1.DrawImage(img0, op); err != nil {
		t.Fatal(err)
		return
	}
	for j := 0; j < img1.Bounds().Size().Y; j++ {
		for i := 0; i < img1.Bounds().Size().X; i++ {
			got := img1.At(i, j).(color.RGBA)
			want := img0.At(i, j).(color.RGBA)
			want.R = uint8(min(0xff, int(want.R)+1))
			want.G = uint8(min(0xff, int(want.G)+2))
			want.B = uint8(min(0xff, int(want.B)+3))
			want.A = uint8(min(0xff, int(want.A)+4))
			if got != want {
				t.Errorf("img1 At(%d, %d): got %#v; want %#v", i, j, got, want)
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
	if _, err := NewImageFromImage(img, FilterNearest); err != nil {
		t.Errorf("NewImageFromImage returns error: %v", err)
	}
}

func TestNewImageFromSubImage(t *testing.T) {
	_, img, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	subImg := img.(*image.NRGBA).SubImage(image.Rect(1, 1, w-1, h-1))
	eimg, err := NewImageFromImage(subImg, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	sw, sh := subImg.Bounds().Dx(), subImg.Bounds().Dy()
	w2, h2 := eimg.Size()
	if w2 != sw {
		t.Errorf("eimg Width: got %#v; want %#v", w2, sw)
	}
	if h2 != sh {
		t.Errorf("eimg Width: got %#v; want %#v", h2, sh)
	}
	for j := 0; j < h2; j++ {
		for i := 0; i < w2; i++ {
			got := eimg.At(i, j)
			want := color.RGBAModel.Convert(img.At(i+1, j+1))
			if got != want {
				t.Errorf("img0 At(%d, %d): got %#v; want %#v", i, j, got, want)
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
	img, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	clr := &mutableRGBA{0x80, 0x80, 0x80, 0x80}
	if err := img.Fill(clr); err != nil {
		t.Fatal(err)
		return
	}
	clr.r = 0
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want := color.RGBA{0x80, 0x80, 0x80, 0x80}
			if got != want {
				t.Errorf("img At(%d, %d): got %#v; want %#v", i, j, got, want)
			}
		}
	}
}

// Issue #740
func TestImageClear(t *testing.T) {
	const w, h = 128, 256
	img, err := NewImage(w, h, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	img.Fill(color.White)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("img At(%d, %d): got %#v; want %#v", i, j, got, want)
			}
		}
	}
	img.Clear()
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want := color.RGBA{}
			if got != want {
				t.Errorf("img At(%d, %d): got %#v; want %#v", i, j, got, want)
			}
		}
	}
}

// Issue #317, #558, #724
func TestImageEdge(t *testing.T) {
	const (
		img0Width        = 16
		img0Height       = 16
		img0InnerWidth   = 10
		img0InnerHeight  = 10
		img0OffsetWidth  = (img0Width - img0InnerWidth) / 2
		img0OffsetHeight = (img0Height - img0InnerHeight) / 2

		img1Width  = 32
		img1Height = 32
	)
	img0, _ := NewImage(img0Width, img0Height, FilterNearest)
	pixels := make([]uint8, 4*img0Width*img0Height)
	for j := 0; j < img0Height; j++ {
		for i := 0; i < img0Width; i++ {
			idx := 4 * (i + j*img0Width)
			switch {
			case img0OffsetWidth <= i && i < img0Width-img0OffsetWidth &&
				img0OffsetHeight <= j && j < img0Height-img0OffsetHeight:
				pixels[idx] = 0xff
				pixels[idx+1] = 0
				pixels[idx+2] = 0
				pixels[idx+3] = 0xff
			default:
				pixels[idx] = 0
				pixels[idx+1] = 0xff
				pixels[idx+2] = 0
				pixels[idx+3] = 0xff
			}
		}
	}
	img0.ReplacePixels(pixels)
	img1, _ := NewImage(img1Width, img1Height, FilterDefault)
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

	img0Sub := img0.SubImage(image.Rect(img0OffsetWidth, img0OffsetHeight, img0Width-img0OffsetWidth, img0Height-img0OffsetHeight)).(*Image)

	for _, s := range []float64{1, 0.5, 0.25} {
		for _, f := range []Filter{FilterNearest, FilterLinear} {
			for _, a := range angles {
				for _, testDrawTriangles := range []bool{false, true} {
					img1.Clear()
					w, h := img0Sub.Size()
					b := img0Sub.Bounds()
					var geo GeoM
					geo.Translate(-float64(w)/2, -float64(h)/2)
					geo.Scale(s, s)
					geo.Rotate(a)
					geo.Translate(img1Width/2, img1Height/2)
					if !testDrawTriangles {
						op := &DrawImageOptions{}
						op.GeoM = geo
						op.Filter = f
						img1.DrawImage(img0Sub, op)
					} else {
						op := &DrawTrianglesOptions{}
						dx0, dy0 := geo.Apply(0, 0)
						dx1, dy1 := geo.Apply(float64(w), 0)
						dx2, dy2 := geo.Apply(0, float64(h))
						dx3, dy3 := geo.Apply(float64(w), float64(h))
						vs := []Vertex{
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
						img1.DrawTriangles(vs, is, img0Sub, op)
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
							case FilterNearest:
								if c == red {
									continue
								}
							case FilterLinear:
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

	src, _ := NewImage(1, 1, FilterNearest)
	dst, _ := NewImage(width, 1, FilterNearest)
	for i := 0; i < width; i++ {
		c := indexToColor(i)
		src.Fill(color.RGBA{c, c, c, 0xff})
		op := &DrawImageOptions{}
		op.GeoM.Translate(float64(i), 0)
		dst.DrawImage(src, op)
	}

	for i := 0; i < width; i++ {
		c := indexToColor(i)
		got := dst.At(i, 0).(color.RGBA)
		want := color.RGBA{c, c, c, 0xff}
		if !sameColors(got, want, 1) {
			t.Errorf("dst.At(%d, %d): got %#v, want: %#v", i, 0, got, want)
		}
	}
}

func BenchmarkDrawImage(b *testing.B) {
	img0, _ := NewImage(16, 16, FilterNearest)
	img1, _ := NewImage(16, 16, FilterNearest)
	op := &DrawImageOptions{}
	for i := 0; i < b.N; i++ {
		img0.DrawImage(img1, op)
	}
}

func TestImageLinearGradiation(t *testing.T) {
	img0, _ := NewImage(2, 2, FilterNearest)
	img0.ReplacePixels([]byte{
		0xff, 0x00, 0x00, 0xff,
		0x00, 0xff, 0x00, 0xff,
		0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	})

	const w, h = 32, 32
	img1, _ := NewImage(w, h, FilterNearest)
	op := &DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(-w/4, -h/4)
	op.Filter = FilterLinear
	img1.DrawImage(img0, op)

	for j := 1; j < h-1; j++ {
		for i := 1; i < w-1; i++ {
			c := img1.At(i, j).(color.RGBA)
			if c.R == 0 || c.R == 0xff {
				t.Errorf("img1.At(%d, %d).R must be in between 0x01 and 0xfe but %#v", i, j, c)
			}
		}
	}
}

func TestImageLinearEdges(t *testing.T) {
	src, _ := NewImage(32, 32, FilterDefault)
	dst, _ := NewImage(64, 64, FilterDefault)
	src.Fill(color.RGBA{0, 0xff, 0, 0xff})
	ebitenutil.DrawRect(src, 8, 8, 16, 16, color.RGBA{0xff, 0, 0, 0xff})

	op := &DrawImageOptions{}
	op.GeoM.Translate(8, 8)
	op.GeoM.Scale(2, 2)
	op.Filter = FilterLinear
	dst.DrawImage(src.SubImage(image.Rect(8, 8, 24, 24)).(*Image), op)

	for j := 0; j < 64; j++ {
		for i := 0; i < 64; i++ {
			c := dst.At(i, j).(color.RGBA)
			got := c.G
			want := uint8(0)
			if abs(int(c.G)-int(want)) > 1 {
				t.Errorf("dst At(%d, %d).G: got %#v, want: %#v", i, j, got, want)
			}
		}
	}
}

func TestImageOutside(t *testing.T) {
	src, _ := NewImage(5, 10, FilterNearest) // internal texture size is 8x16.
	dst, _ := NewImage(4, 4, FilterNearest)
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

		op := &DrawImageOptions{}
		op.GeoM.Translate(0, 0)
		dst.DrawImage(src.SubImage(image.Rectangle{
			Min: image.Pt(c.X, c.Y),
			Max: image.Pt(c.X+c.Width, c.Y+c.Height),
		}).(*Image), op)

		for j := 0; j < 4; j++ {
			for i := 0; i < 4; i++ {
				got := dst.At(i, j).(color.RGBA)
				want := color.RGBA{0, 0, 0, 0}
				if got != want {
					t.Errorf("src(x: %d, y: %d, w: %d, h: %d), dst At(%d, %d): got %#v, want: %#v", c.X, c.Y, c.Width, c.Height, i, j, got, want)
				}
			}
		}
	}
}

func TestImageOutsideUpperLeft(t *testing.T) {
	src, _ := NewImage(4, 4, FilterNearest)
	dst1, _ := NewImage(16, 16, FilterNearest)
	dst2, _ := NewImage(16, 16, FilterNearest)
	src.Fill(color.RGBA{0xff, 0, 0, 0xff})

	op := &DrawImageOptions{}
	op.GeoM.Rotate(math.Pi / 4)
	dst1.DrawImage(src.SubImage(image.Rect(-4, -4, 8, 8)).(*Image), op)

	op = &DrawImageOptions{}
	// The outside part of the source rect is just ignored.
	// This behavior was changed as of 1.9.0-alpha.
	// op.GeoM.Translate(4, 4)
	op.GeoM.Rotate(math.Pi / 4)
	dst2.DrawImage(src, op)

	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			got := dst1.At(i, j).(color.RGBA)
			want := dst2.At(i, j).(color.RGBA)
			if got != want {
				t.Errorf("got: dst1.At(%d, %d): %#v, want: dst2.At(%d, %d): %#v", i, j, got, i, j, want)
			}
		}
	}
}

func TestImageSize(t *testing.T) {
	const (
		w = 17
		h = 31
	)
	img, _ := NewImage(w, h, FilterDefault)
	gotW, gotH := img.Size()
	if gotW != w {
		t.Errorf("got: %d, want: %d", gotW, w)
	}
	if gotH != h {
		t.Errorf("got: %d, want: %d", gotH, h)
	}
}

func TestImageSize1(t *testing.T) {
	src, _ := NewImage(1, 1, FilterNearest)
	dst, _ := NewImage(1, 1, FilterNearest)
	src.Fill(color.White)
	dst.DrawImage(src, nil)
	got := src.At(0, 0).(color.RGBA)
	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	if !sameColors(got, want, 1) {
		t.Errorf("got: %#v, want: %#v", got, want)
	}
}

func TestImageSize4096(t *testing.T) {
	src, _ := NewImage(4096, 4096, FilterNearest)
	dst, _ := NewImage(4096, 4096, FilterNearest)
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
	src.ReplacePixels(pix)
	dst.DrawImage(src, nil)
	for i := 4095; i < 4096; i++ {
		j := 4095
		got := dst.At(i, j).(color.RGBA)
		want := color.RGBA{uint8(i + j), uint8((i + j) >> 8), uint8((i + j) >> 16), 0xff}
		if got != want {
			t.Errorf("At(%d, %d): got: %#v, want: %#v", i, j, got, want)
		}
	}
	for j := 4095; j < 4096; j++ {
		i := 4095
		got := dst.At(i, j).(color.RGBA)
		want := color.RGBA{uint8(i + j), uint8((i + j) >> 8), uint8((i + j) >> 16), 0xff}
		if got != want {
			t.Errorf("At(%d, %d): got: %#v, want: %#v", i, j, got, want)
		}
	}
}

func TestImageCopy(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("copying image and using it must panic")
		}
	}()

	img0, _ := NewImage(256, 256, FilterDefault)
	img1 := *img0
	img1.Fill(color.Transparent)
}

// Issue #611, #907
func TestImageStretch(t *testing.T) {
	const w = 16

	dst, _ := NewImage(w, 4096, FilterDefault)
loop:
	for h := 1; h <= 32; h++ {
		src, _ := NewImage(w, h+1, FilterDefault)

		pix := make([]byte, 4*w*(h+1))
		for i := 0; i < w*h; i++ {
			pix[4*i] = 0xff
			pix[4*i+3] = 0xff
		}
		for i := 0; i < w; i++ {
			pix[4*(w*h+i)+1] = 0xff
			pix[4*(w*h+i)+3] = 0xff
		}
		src.ReplacePixels(pix)

		_, dh := dst.Size()
		for i := 0; i < dh; {
			dst.Clear()
			op := &DrawImageOptions{}
			op.GeoM.Scale(1, float64(i)/float64(h))
			dst.DrawImage(src.SubImage(image.Rect(0, 0, w, h)).(*Image), op)
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
					t.Errorf("At(%d, %d) (height=%d, scale=%d/%d): got: %#v, want: %#v", 0, i+j, h, i, h, got, want)
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

	src, _ := NewImage(4, 4, FilterNearest)
	src.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	dst, _ := NewImage(width, height, FilterNearest)
	for j := 0; j < height/4; j++ {
		for i := 0; i < width/4; i++ {
			op := &DrawImageOptions{}
			op.GeoM.Translate(float64(i*4), float64(j*4))
			dst.DrawImage(src, op)
		}
	}

	for j := 0; j < height/4; j++ {
		for i := 0; i < width/4; i++ {
			got := dst.At(i*4, j*4).(color.RGBA)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got %#v, want: %#v", i*4, j*4, got, want)
			}
		}
	}
}

func TestImageMipmap(t *testing.T) {
	src, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := src.Size()

	l1, _ := NewImage(w/2, h/2, FilterDefault)
	op := &DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = FilterLinear
	l1.DrawImage(src, op)

	l1w, l1h := l1.Size()
	l2, _ := NewImage(l1w/2, l1h/2, FilterDefault)
	op = &DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = FilterLinear
	l2.DrawImage(l1, op)

	gotDst, _ := NewImage(w, h, FilterDefault)
	op = &DrawImageOptions{}
	op.GeoM.Scale(1/5.0, 1/5.0)
	op.Filter = FilterLinear
	gotDst.DrawImage(src, op)

	wantDst, _ := NewImage(w, h, FilterDefault)
	op = &DrawImageOptions{}
	op.GeoM.Scale(4.0/5.0, 4.0/5.0)
	op.Filter = FilterLinear
	wantDst.DrawImage(l2, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := gotDst.At(i, j).(color.RGBA)
			want := wantDst.At(i, j).(color.RGBA)
			if !sameColors(got, want, 1) {
				t.Errorf("At(%d, %d): got: %#v, want: %#v", i, j, got, want)
			}
		}
	}
}

func TestImageMipmapNegativeDet(t *testing.T) {
	src, _, err := openEbitenImage()
	if err != nil {
		t.Fatal(err)
		return
	}
	w, h := src.Size()

	l1, _ := NewImage(w/2, h/2, FilterDefault)
	op := &DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = FilterLinear
	l1.DrawImage(src, op)

	l1w, l1h := l1.Size()
	l2, _ := NewImage(l1w/2, l1h/2, FilterDefault)
	op = &DrawImageOptions{}
	op.GeoM.Scale(1/2.0, 1/2.0)
	op.Filter = FilterLinear
	l2.DrawImage(l1, op)

	gotDst, _ := NewImage(w, h, FilterDefault)
	op = &DrawImageOptions{}
	op.GeoM.Scale(-1/5.0, -1/5.0)
	op.GeoM.Translate(float64(w), float64(h))
	op.Filter = FilterLinear
	gotDst.DrawImage(src, op)

	wantDst, _ := NewImage(w, h, FilterDefault)
	op = &DrawImageOptions{}
	op.GeoM.Scale(-4.0/5.0, -4.0/5.0)
	op.GeoM.Translate(float64(w), float64(h))
	op.Filter = FilterLinear
	wantDst.DrawImage(l2, op)

	allZero := true
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := gotDst.At(i, j).(color.RGBA)
			want := wantDst.At(i, j).(color.RGBA)
			if !sameColors(got, want, 1) {
				t.Errorf("At(%d, %d): got: %#v, want: %#v", i, j, got, want)
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
	img0, _ := NewImage(256, 256, FilterDefault)
	img1, _ := NewImage(128, 128, FilterDefault)
	img1.Fill(color.White)

	for i := 0; i < 8; i++ {
		img0.Clear()

		s := 1 - float64(i)/8

		op := &DrawImageOptions{}
		op.Filter = FilterLinear
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
			t.Errorf("want: %#v, got: %#v", want, got)
		}
	}
}

// Issue #725
func TestImageMiamapAndDrawTriangle(t *testing.T) {
	img0, _ := NewImage(32, 32, FilterDefault)
	img1, _ := NewImage(128, 128, FilterDefault)
	img2, _ := NewImage(128, 128, FilterDefault)

	// Fill img1 red and create img1's mipmap
	img1.Fill(color.RGBA{0xff, 0, 0, 0xff})
	op := &DrawImageOptions{}
	op.GeoM.Scale(0.25, 0.25)
	op.Filter = FilterLinear
	img0.DrawImage(img1, op)

	// Call DrawTriangle on img1 and fill it with green
	img2.Fill(color.RGBA{0, 0xff, 0, 0xff})
	vs := []Vertex{
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
	op = &DrawImageOptions{}
	op.GeoM.Scale(0.25, 0.25)
	op.Filter = FilterLinear
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
	img, _ := NewImage(16, 16, FilterDefault)
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
	img, _ := NewImage(16, 16, FilterDefault)
	img.Fill(color.RGBA{0xff, 0, 0, 0xff})

	got, _ := img.SubImage(image.Rect(1, 1, 16, 16)).(*Image).Size()
	want := 15
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageDrawImmediately(t *testing.T) {
	const w, h = 16, 16
	img0, _ := NewImage(w, h, FilterDefault)
	img1, _ := NewImage(w, h, FilterDefault)
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
	src, _ := NewImage(w, h, FilterDefault)
	dst, _ := NewImage(int(math.Floor(w*scale)), h, FilterDefault)

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
	src.ReplacePixels(pix)

	for _, f := range []Filter{FilterNearest, FilterLinear} {
		op := &DrawImageOptions{}
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
					t.Errorf("src.At(%d, %d): filter: %d, got: %v, want: %v", i, j, f, got, want)
				}
			}
		}
	}
}

func TestImageAddressRepeat(t *testing.T) {
	const w, h = 16, 16
	src, _ := NewImage(w, h, FilterDefault)
	dst, _ := NewImage(w, h, FilterDefault)
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
	src.ReplacePixels(pix)

	vs := []Vertex{
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
	op := &DrawTrianglesOptions{}
	op.Address = AddressRepeat
	dst.DrawTriangles(vs, is, src.SubImage(image.Rect(4, 4, 8, 8)).(*Image), op)

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

func TestImageReplacePixelsAfterClear(t *testing.T) {
	const w, h = 256, 256
	img, _ := NewImage(w, h, FilterDefault)
	img.ReplacePixels(make([]byte, 4*w*h))
	// Clear used to call DrawImage to clear the image, which was the cause of crash. It is because after
	// DrawImage is called, ReplacePixels for a region is forbidden.
	//
	// Now ReplacePixels was always called at Clear instead.
	img.Clear()
	img.ReplacePixels(make([]byte, 4*w*h))

	// The test passes if this doesn't crash.
}

func TestImageSet(t *testing.T) {
	type Pt struct {
		X, Y int
	}

	const w, h = 16, 16
	img, _ := NewImage(w, h, FilterDefault)
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
	src, _ := NewImage(w, h, FilterDefault)
	dst, _ := NewImage(w, h, FilterDefault)
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
	op := &DrawImageOptions{}
	op.GeoM.Translate(2, 2)
	dst.DrawImage(src.SubImage(image.Rect(2, 2, w-2, h-2)).(*Image), op)
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
	src0, _ := NewImage(w, h, FilterDefault)
	src1, _ := NewImage(w, h, FilterDefault)
	dst0, _ := NewImage(w, h, FilterDefault)
	dst1, _ := NewImage(w, h, FilterDefault)

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
	src0.ReplacePixels(pix0)

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
	src1.ReplacePixels(pix1)

	dst0.Fill(color.Black)
	dst1.Fill(color.Black)

	op := &DrawImageOptions{}
	op.GeoM.Scale(0.5, 0.5)
	op.Filter = FilterLinear
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
	src, _ := NewImage(w, h, FilterDefault)
	dst, _ := NewImage(w, h, FilterDefault)

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
	src.ReplacePixels(pix)

	vs := []Vertex{
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
	op := &DrawTrianglesOptions{}
	dst.DrawTriangles(vs, is, src.SubImage(image.Rect(4, 4, 8, 8)).(*Image), op)

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
	img, _ := NewImage(16, 16, FilterDefault)
	img.Set(0, 0, color.White)
	img.SubImage(image.Rect(0, 0, 16, 16))
	runtime.GC()
	got := img.At(0, 0)
	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	img.Set(0, 1, color.White)
	sub := img.SubImage(image.Rect(0, 0, 16, 16)).(*Image)
	sub.Dispose()
	got = img.At(0, 1)
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageSubImageSubImage(t *testing.T) {
	img, _ := NewImage(16, 16, FilterDefault)
	img.Fill(color.White)
	sub0 := img.SubImage(image.Rect(0, 0, 12, 12)).(*Image)
	sub1 := sub0.SubImage(image.Rect(4, 4, 16, 16)).(*Image)
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
	src, _ := NewImage(w, h, FilterDefault)
	dst, _ := NewImage(w, h, FilterDefault)

	src.Fill(color.White)
	op := &DrawImageOptions{}
	op.GeoM.Scale(1, 0.24)
	op.Filter = FilterLinear
	dst.DrawImage(src.SubImage(image.Rect(5, 0, 6, 16)).(*Image), op)
	got := dst.At(0, 0).(color.RGBA)
	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageZeroSizedMipmap(t *testing.T) {
	const w, h = 16, 16
	src, _ := NewImage(w, h, FilterDefault)
	dst, _ := NewImage(w, h, FilterDefault)

	op := &DrawImageOptions{}
	op.Filter = FilterLinear
	dst.DrawImage(src.SubImage(image.ZR).(*Image), op)
}

// Issue #898
func TestImageFillingAndEdges(t *testing.T) {
	const (
		srcw, srch = 16, 16
		dstw, dsth = 256, 16
	)

	src, _ := NewImage(srcw, srch, FilterDefault)
	dst, _ := NewImage(dstw, dsth, FilterDefault)

	src.Fill(color.White)
	dst.Fill(color.Black)

	op := &DrawImageOptions{}
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
	dst, _ := NewImage(w, h, FilterDefault)
	src, _ := NewImage(w, h, FilterDefault)
	clr := color.RGBA{0xff, 0, 0, 0xff}
	src.Fill(clr)

	vs := []Vertex{
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
