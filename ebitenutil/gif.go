/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebitenutil

import (
	"github.com/hajimehoshi/ebiten"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"io"
)

type recorder struct {
	inner        func(screen *ebiten.Image) error
	writer       io.Writer
	gif          *gif.GIF
	currentFrame int
}

func (r *recorder) update(screen *ebiten.Image) error {
	if err := r.inner(screen); err != nil {
		return err
	}
	if r.currentFrame == len(r.gif.Image) {
		return nil
	}
	img := image.NewPaletted(screen.Bounds(), palette.Plan9)
	draw.Draw(img, img.Bounds(), screen, screen.Bounds().Min, draw.Over)
	r.gif.Image[r.currentFrame] = img
	// The actual FPS is 60, but GIF can't have such FPS. Set 50 FPS instead.
	r.gif.Delay[r.currentFrame] = 2

	r.currentFrame++
	if r.currentFrame == len(r.gif.Image) {
		if err := gif.EncodeAll(r.writer, r.gif); err != nil {
			return err
		}
	}
	return nil
}

// RecordScreenAsGIF returns updating function with recording the screen as an animation GIF image.
//
// This encodes each screen at each frame and may slows the application.
func RecordScreenAsGIF(update func(*ebiten.Image) error, out io.Writer, frameNum int) func(*ebiten.Image) error {
	r := &recorder{
		inner:  update,
		writer: out,
		gif: &gif.GIF{
			Image:     make([]*image.Paletted, frameNum),
			Delay:     make([]int, frameNum),
			LoopCount: -1,
		},
	}
	return r.update
}
