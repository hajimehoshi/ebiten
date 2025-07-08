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

//go:build ignore

package main

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	init  bool
	count int
	end0  chan struct{}
	end1  chan struct{}
	errCh chan error
}

func (g *Game) Update() error {
	if !g.init {
		end0 := make(chan struct{})
		end1 := make(chan struct{})
		errCh := make(chan error)
		src := ebiten.NewImage(1, 2)
		src.WritePixels([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
		img := ebiten.NewImage(1, 2)
		go func() {
			t := time.Tick(time.Microsecond)
		loop:
			for {
				select {
				case <-t:
					// Call DrawImage every time in order to invalidate the internal pixels cache.
					img.DrawImage(src, nil)
					got := img.At(0, 0).(color.RGBA)
					want := color.RGBA{0xff, 0xff, 0xff, 0xff}
					if got != want {
						errCh <- fmt.Errorf("got: %v, want: %v", got, want)
						close(errCh)
						return
					}
				case <-end0:
					close(end1)
					break loop
				}
			}
		}()
		g.end0 = end0
		g.end1 = end1
		g.errCh = errCh
		g.init = true
	}

	select {
	case err := <-g.errCh:
		return err
	default:
	}

	g.count++
	if g.count >= 60 {
		if g.end0 != nil {
			close(g.end0)
			g.end0 = nil
		}
		select {
		case <-g.end1:
			return ebiten.Termination
		default:
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(w, h int) (int, int) {
	return 320, 240
}

func main() {
	if err := ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
