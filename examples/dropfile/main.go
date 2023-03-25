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

package main

import (
	"image/png"
	"io/fs"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	images []*ebiten.Image
	err    error

	m sync.Mutex
}

func (g *Game) Update() error {
	if err := func() error {
		g.m.Lock()
		defer g.m.Unlock()
		return g.err
	}(); err != nil {
		return err
	}

	if files := ebiten.DroppedFiles(); files != nil {
		go func() {
			if err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				fi, err := d.Info()
				if err != nil {
					return err
				}
				log.Printf("Name: %s, Size: %d, IsDir: %t, ModTime: %v", fi.Name(), fi.Size(), fi.IsDir(), fi.ModTime())

				f, err := files.Open(path)
				if err != nil {
					return err
				}
				defer func() {
					_ = f.Close()
				}()

				img, err := png.Decode(f)
				if err != nil {
					return nil
				}
				eimg := ebiten.NewImageFromImage(img)

				g.m.Lock()
				g.images = append(g.images, eimg)
				g.m.Unlock()

				return nil
			}); err != nil {
				g.m.Lock()
				if g.err == nil {
					g.err = err
				}
				g.m.Unlock()
			}
		}()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.m.Lock()
	defer g.m.Unlock()

	if len(g.images) == 0 {
		ebitenutil.DebugPrint(screen, "Drop PNG files onto this window!")
		return
	}

	const imageSize = 128
	xcount := screen.Bounds().Dx() / imageSize
	if xcount == 0 {
		return
	}

	for i, img := range g.images {
		x := (i % xcount) * imageSize
		y := (i / xcount) * imageSize

		s := imageSize / float64(img.Bounds().Dx())
		if sy := imageSize / float64(img.Bounds().Dy()); s > sy {
			s = sy
		}
		if s > 1 {
			s = 1
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(float64(x), float64(y))
		op.Filter = ebiten.FilterLinear
		screen.DrawImage(img, op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Dropping Files (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
