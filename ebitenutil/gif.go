// Copyright 2014 Hajime Hoshi
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

package ebitenutil

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"io"
	"sync"

	"github.com/hajimehoshi/ebiten"
)

type recorder struct {
	inner        func(screen *ebiten.Image) error
	writer       io.Writer
	frameNum     int
	skips        int
	gif          *gif.GIF
	currentFrame int
	wg           sync.WaitGroup
}

var cheapPalette color.Palette

func init() {
	cs := []color.Color{}
	for _, r := range []uint8{0x00, 0x80, 0xff} {
		for _, g := range []uint8{0x00, 0x80, 0xff} {
			for _, b := range []uint8{0x00, 0x80, 0xff} {
				cs = append(cs, color.RGBA{r, g, b, 0xff})
			}
		}
	}
	cheapPalette = color.Palette(cs)
}

func (r *recorder) delay() int {
	delay := 100 * r.skips / ebiten.MaxTPS()
	if delay < 2 {
		return 2
	}
	return delay
}

func (r *recorder) palette() color.Palette {
	if 1 < (r.frameNum-1)/r.skips+1 {
		return cheapPalette
	}
	return palette.Plan9
}

func (r *recorder) update(screen *ebiten.Image) error {
	if err := r.inner(screen); err != nil {
		return err
	}
	if r.currentFrame == r.frameNum {
		return nil
	}
	if r.currentFrame%r.skips == 0 {
		if r.gif == nil {
			num := (r.frameNum-1)/r.skips + 1
			r.gif = &gif.GIF{
				Image:     make([]*image.Paletted, num),
				Delay:     make([]int, num),
				LoopCount: -1,
			}
		}
		s := image.NewNRGBA(screen.Bounds())
		draw.Draw(s, s.Bounds(), screen, screen.Bounds().Min, draw.Src)

		img := image.NewPaletted(s.Bounds(), r.palette())
		f := r.currentFrame / r.skips
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			draw.FloydSteinberg.Draw(img, img.Bounds(), s, s.Bounds().Min)
			r.gif.Image[f] = img
			r.gif.Delay[f] = r.delay()
		}()
	}

	r.currentFrame++
	if r.currentFrame == r.frameNum {
		r.wg.Wait()
		if err := gif.EncodeAll(r.writer, r.gif); err != nil {
			return err
		}
	}
	return nil
}

// RecordScreenAsGIF returns updating function with recording the screen as an animation GIF image.
//
// Deprecated: (as of 1.6.0) Do not use this.
//
// This encodes each screen at each frame and may slows the application.
//
// Here is the example to record initial 120 frames of your game:
//
//     func update(screen *ebiten.Image) error {
//         // ...
//     }
//
//     func main() {
//         out, err := os.Create("output.gif")
//         if err != nil {
//             log.Fatal(err)
//         }
//         defer out.Close()
//
//         update := RecordScreenAsGIF(update, out, 120)
//         if err := ebiten.Run(update, 320, 240, 2, "Your game's title"); err != nil {
//             log.Fatal(err)
//         }
//     }
func RecordScreenAsGIF(update func(*ebiten.Image) error, out io.Writer, frameNum int) func(*ebiten.Image) error {
	r := &recorder{
		inner:    update,
		writer:   out,
		frameNum: frameNum,
		skips:    10,
	}
	return r.update
}
