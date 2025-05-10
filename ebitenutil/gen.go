// Copyright 2022 The Ebitengine Authors
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

//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/hajimehoshi/bitmapfont/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

func run() error {
	// These values are copied from an example in github.com/hajimehoshi/bitmapfont.
	const (
		charWidth  = 6
		lineHeight = 16
	)

	var lines []string
	for j := 0; j < 8; j++ {
		var line string
		for i := 0; i < 32; i++ {
			line += string(rune(i + j*32))
		}
		lines = append(lines, line)
	}

	dst := image.NewRGBA(image.Rect(0, 0, charWidth*32, lineHeight*8))
	for i, clr := range []color.Color{color.RGBA{0, 0, 0, 0x80}, color.White} {
		var offsetX int
		var offsetY int
		if i == 0 {
			offsetX = 1
			offsetY = 1
		}
		d := font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(clr),
			Face: bitmapfont.Face,
			Dot:  fixed.Point26_6{X: fixed.I(offsetX), Y: bitmapfont.Face.Metrics().Ascent + fixed.I(offsetY)},
		}
		for _, line := range lines {
			d.Dot.X = fixed.I(offsetX)
			d.DrawString(line)
			d.Dot.Y += fixed.I(lineHeight)
		}
	}

	f, err := os.Create("text.png")
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if err := png.Encode(w, dst); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}
