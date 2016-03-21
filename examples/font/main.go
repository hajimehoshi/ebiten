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

package main

import (
	"image"
	"io/ioutil"
	"log"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	textImage *ebiten.Image
)

// text is a head part of a Japanese novel 山月記 (Sangetsuki)
// See http://www.aozora.gr.jp/cards/000119/files/624_14544.html.
var text = []string{
	"隴西の李徴は博学才穎、天宝の末年、",
	"若くして名を虎榜に連ね、",
	"ついで江南尉に補せられたが、",
	"性、狷介、自ら恃むところ頗厚く、",
	"賤吏に甘んずるを潔しとしなかった。",
}

func parseFont() error {
	f, err := ebitenutil.OpenFile("_resources/fonts/mplus-1p-regular.ttf")
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	tt, err := truetype.Parse(b)
	if err != nil {
		return err
	}
	w, h := textImage.Size()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	const size = 24
	const dpi = 72
	d := &font.Drawer{
		Dst: dst,
		Src: image.White,
		Face: truetype.NewFace(tt, &truetype.Options{
			Size:    size,
			DPI:     dpi,
			Hinting: font.HintingFull,
		}),
	}
	dy := size * dpi / 72
	y := dy
	for _, s := range text {
		d.Dot = fixed.P(0, y)
		d.DrawString(s)
		y += dy
	}
	textImage.ReplacePixels(dst.Pix)
	return nil
}

func update(screen *ebiten.Image) error {
	if err := screen.DrawImage(textImage, &ebiten.DrawImageOptions{}); err != nil {
		return err
	}
	return nil
}

func main() {
	var err error
	textImage, err = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := parseFont(); err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Font (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
