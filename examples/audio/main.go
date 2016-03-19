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
	"fmt"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/exp/audio"
	"github.com/hajimehoshi/ebiten/exp/audio/vorbis"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	playerBarImage     *ebiten.Image
	playerCurrentImage *ebiten.Image
)

func init() {
	var err error
	playerBarImage, err = ebiten.NewImage(300, 4, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	playerBarImage.Fill(&color.RGBA{0x80, 0x80, 0x80, 0xff})

	playerCurrentImage, err = ebiten.NewImage(4, 10, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	playerCurrentImage.Fill(&color.RGBA{0xff, 0xff, 0xff, 0xff})
}

var (
	audioContext     *audio.Context
	audioLoadingDone chan struct{}
	audioLoaded      bool
	audioPlayer      *audio.Player
	total            time.Duration
	mouseButtonState = map[ebiten.MouseButton]int{}
)

func playerBarRect() (x, y, w, h int) {
	w, h = playerBarImage.Size()
	x = (screenWidth - w) / 2
	y = screenHeight - h - 16
	return
}

func updatePlayerBar() error {
	if !audioLoaded {
		return nil
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		mouseButtonState[ebiten.MouseButtonLeft] = 0
		return nil
	}
	mouseButtonState[ebiten.MouseButtonLeft]++
	if mouseButtonState[ebiten.MouseButtonLeft] != 1 {
		return nil
	}
	x, y := ebiten.CursorPosition()
	bx, by, bw, bh := playerBarRect()
	if y < by || by+bh <= y {
		return nil
	}
	if x < bx || bx+bw <= x {
		return nil
	}
	p := time.Duration(x-bx) * total / time.Duration(bw)
	return audioPlayer.Seek(p)
}

func update(screen *ebiten.Image) error {
	audioContext.Update()
	if !audioLoaded {
		select {
		case <-audioLoadingDone:
			audioLoaded = true
		default:
		}
	}

	if err := updatePlayerBar(); err != nil {
		return err
	}

	op := &ebiten.DrawImageOptions{}
	x, y, w, h := playerBarRect()
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(playerBarImage, op)
	currentTimeStr := ""
	if audioLoaded && audioPlayer.IsPlaying() {
		c := audioPlayer.Current()

		// Current Time
		m := (c / time.Minute) % 100
		s := (c / time.Second) % 60
		currentTimeStr = fmt.Sprintf("%02d:%02d", m, s)

		// Bar
		cw, ch := playerCurrentImage.Size()
		cx := int(time.Duration(w)*c/total) + x - cw/2
		cy := y - (ch-h)/2
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(cx), float64(cy))
		screen.DrawImage(playerCurrentImage, op)
	}

	msg := fmt.Sprintf(`FPS: %0.2f
%s`, ebiten.CurrentFPS(), currentTimeStr)
	if !audioLoaded {
		msg += "\nNow Loading..."
	}
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	// Use a FLAC file so far: I couldn't find any good OGG/Vorbis decoder in pure Go.
	f, err := ebitenutil.OpenFile("_resources/audio/ragtime.ogg")
	if err != nil {
		log.Fatal(err)
	}
	// TODO: sampleRate should be obtained from the ogg file.
	audioContext = audio.NewContext(22050)
	audioLoadingDone = make(chan struct{})
	// TODO: This doesn't work synchronously on browsers because of decoding. Fix this.
	go func() {
		var err error
		s, err := vorbis.Decode(audioContext, f)
		if err != nil {
			log.Fatal(err)
			return
		}
		total = s.Len()
		audioPlayer, err = audioContext.NewPlayer(s)
		if err != nil {
			log.Fatal(err)
			return
		}
		close(audioLoadingDone)
		audioPlayer.Play()
	}()
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Audio (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
