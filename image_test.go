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
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math"
	"os"
	"testing"

	. "github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

func TestMain(m *testing.M) {
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

var ebitenImageBin = ""

func openImage(path string) (image.Image, error) {
	file, err := readFile(path)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func openEbitenImage(path string) (*Image, image.Image, error) {
	img, err := openImage(path)
	if err != nil {
		return nil, nil, err
	}

	eimg, err := NewImageFromImage(img, FilterNearest)
	if err != nil {
		return nil, nil, err
	}
	return eimg, img, nil
}

func diff(x, y uint8) uint8 {
	if x <= y {
		return y - x
	}
	return x - y
}

func TestImagePixels(t *testing.T) {
	img0, img, err := openEbitenImage("testdata/ebiten.png")
	if err != nil {
		t.Fatal(err)
		return
	}

	if got := img0.Bounds().Size(); got != img.Bounds().Size() {
		t.Fatalf("img size: got %d; want %d", got, img.Bounds().Size())
	}

	w, h := img0.Bounds().Size().X, img0.Bounds().Size().Y
	// Check out of range part
	w2, h2 := graphics.NextPowerOf2Int(w), graphics.NextPowerOf2Int(h)
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
	img1, _, err := openEbitenImage("testdata/ebiten.png")
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
			if 1 < diff(c1.R, c2.R) || 1 < diff(c1.G, c2.G) || 1 < diff(c1.B, c2.B) || 1 < diff(c1.A, c2.A) {
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
	img, _, err := openEbitenImage("testdata/ebiten.png")
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := img.DrawImage(img, nil); err == nil {
		t.Fatalf("img.DrawImage(img, nil) doesn't return error; an error should be returned")
	}
}

func TestImageScale(t *testing.T) {
	for _, scale := range []int{2, 3, 4} {
		img0, _, err := openEbitenImage("testdata/ebiten.png")
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

		for j := 0; j < h*2; j++ {
			for i := 0; i < w*2; i++ {
				c0 := img0.At(i/scale, j/scale).(color.RGBA)
				c1 := img1.At(i, j).(color.RGBA)
				if c0 != c1 {
					t.Errorf("img0.At(%[1]d, %[2]d) should equal to img1.At(%[3]d, %[4]d) (with scale %[5]d) but not: %[6]v vs %[7]v", i/2, j/2, i, j, scale, c0, c1)
				}
			}
		}
	}
}

func TestImage90DegreeRotate(t *testing.T) {
	img0, _, err := openEbitenImage("testdata/ebiten.png")
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
	img0, _, err := openEbitenImage("testdata/ebiten.png")
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

func TestReplacePixels(t *testing.T) {
	origImg, err := openImage("testdata/ebiten.png")
	if err != nil {
		t.Fatal(err)
		return
	}
	// Convert to RGBA
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

func TestImageDispose(t *testing.T) {
	img, err := NewImage(16, 16, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := img.Dispose(); err != nil {
		t.Errorf("img.Dipose() returns error: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestImageCompositeModeLighter(t *testing.T) {
	img0, _, err := openEbitenImage("testdata/ebiten.png")
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
	img, _, err := openEbitenImage("testdata/ebiten.png")
	if err != nil {
		t.Fatal(err)
		return
	}
	if _, err := NewImageFromImage(img, FilterNearest); err != nil {
		t.Errorf("NewImageFromImage returns error: %v", err)
	}
}

func TestNewImageFromSubImage(t *testing.T) {
	img, err := openImage("testdata/ebiten.png")
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

type halfImagePart struct {
	image *Image
}

func (p *halfImagePart) Len() int {
	return 1
}

func (p *halfImagePart) Src(index int) (int, int, int, int) {
	w, h := p.image.Size()
	return 0, 0, w, h / 2
}

func (p *halfImagePart) Dst(index int) (int, int, int, int) {
	w, h := p.image.Size()
	return 0, 0, w, h / 2
}

// Issue 317
func TestImageEdge(t *testing.T) {
	const (
		img0Width  = 16
		img0Height = 16
		img1Width  = 32
		img1Height = 32
	)
	img0, err := NewImage(img0Width, img0Height, FilterNearest)
	if err != nil {
		t.Fatal(err)
	}
	pixels := make([]uint8, 4*img0Width*img0Height)
	for j := 0; j < img0Height; j++ {
		for i := 0; i < img0Width; i++ {
			idx := 4 * (i + j*img0Width)
			switch {
			case j < img0Height/2:
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
	if err := img0.ReplacePixels(pixels); err != nil {
		t.Fatal(err)
	}
	img1, err := NewImage(img1Width, img1Height, FilterNearest)
	if err != nil {
		t.Fatal(err)
	}
	red := color.RGBA{0xff, 0, 0, 0xff}
	transparent := color.RGBA{0, 0, 0, 0}
	// Unfortunately, TravisCI couldn't pass this test for some angles.
	// https://travis-ci.org/hajimehoshi/ebiten/builds/200454658
	// Let's use 'decent' angles here.
	for _, a := range []int{0, 45, 90, 135, 180, 225, 270, 315, 360} {
		if err := img1.Clear(); err != nil {
			t.Fatal(err)
		}
		op := &DrawImageOptions{}
		op.ImageParts = &halfImagePart{img0}
		op.GeoM.Translate(-float64(img0Width)/2, -float64(img0Height)/2)
		op.GeoM.Rotate(float64(a) * math.Pi / 180)
		op.GeoM.Translate(img1Width/2, img1Height/2)
		if err := img1.DrawImage(img0, op); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < img1Height; j++ {
			for i := 0; i < img1Width; i++ {
				c := img1.At(i, j)
				if c == red {
					continue
				}
				if c == transparent {
					continue
				}
				t.Errorf("img1.At(%d, %d) (angle: %d) want: red or transparent, got: %v", i, j, a, c)
			}
		}
	}
}
