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
	"bytes"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/exp/audio"
	"github.com/hajimehoshi/ebiten/exp/audio/vorbis"
	"github.com/hajimehoshi/ebiten/exp/audio/wav"
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

type Player struct {
	audioPlayer *audio.Player
	total       time.Duration
}

var (
	audioContext     *audio.Context
	musicPlayer      *Player
	seStream         *wav.Stream
	seBuffer         []byte
	musicCh          = make(chan *Player)
	seCh             = make(chan *wav.Stream)
	mouseButtonState = map[ebiten.MouseButton]int{}
	keyState         = map[ebiten.Key]int{}
	volume128        = 128
)

func playerBarRect() (x, y, w, h int) {
	w, h = playerBarImage.Size()
	x = (screenWidth - w) / 2
	y = screenHeight - h - 16
	return
}

func (p *Player) updateSE() error {
	if seStream == nil {
		return nil
	}
	if !ebiten.IsKeyPressed(ebiten.KeyP) {
		keyState[ebiten.KeyP] = 0
		return nil
	}
	keyState[ebiten.KeyP]++
	if keyState[ebiten.KeyP] != 1 {
		return nil
	}
	// Clone the buffer so that we can play the same SE mutiple times.
	// TODO(hajimehoshi): This consumes memory. Can we avoid this?
	if seBuffer == nil {
		b, err := ioutil.ReadAll(seStream)
		if err != nil {
			return err
		}
		seBuffer = b
	}
	sePlayer, err := audioContext.NewPlayer(bytes.NewReader(seBuffer))
	if err != nil {
		return err
	}
	return sePlayer.Play()
}

func (p *Player) updateVolume() error {
	if p.audioPlayer == nil {
		return nil
	}
	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		volume128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		volume128++
	}
	if volume128 < 0 {
		volume128 = 0
	}
	if 128 < volume128 {
		volume128 = 128
	}
	p.audioPlayer.SetVolume(float64(volume128) / 128)
	return nil
}

func (p *Player) updatePlayPause() error {
	if p.audioPlayer == nil {
		return nil
	}
	if !ebiten.IsKeyPressed(ebiten.KeyS) {
		keyState[ebiten.KeyS] = 0
		return nil
	}
	keyState[ebiten.KeyS]++
	if keyState[ebiten.KeyS] != 1 {
		return nil
	}
	if p.audioPlayer.IsPlaying() {
		return p.audioPlayer.Pause()
	}
	return p.audioPlayer.Play()
}

func (p *Player) updateBar() error {
	if p.audioPlayer == nil {
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
	pos := time.Duration(x-bx) * p.total / time.Duration(bw)
	return p.audioPlayer.Seek(pos)
}

func update(screen *ebiten.Image) error {
	audioContext.Update()
	if musicPlayer == nil {
		select {
		case musicPlayer = <-musicCh:
		default:
		}
	}
	if seStream == nil {
		select {
		case seStream = <-seCh:
		default:
		}
	}
	if musicPlayer != nil {
		if err := musicPlayer.updateBar(); err != nil {
			return err
		}
		if err := musicPlayer.updatePlayPause(); err != nil {
			return err
		}
		if err := musicPlayer.updateSE(); err != nil {
			return err
		}
		if err := musicPlayer.updateVolume(); err != nil {
			return err
		}
	}

	op := &ebiten.DrawImageOptions{}
	x, y, w, h := playerBarRect()
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(playerBarImage, op)
	currentTimeStr := "00:00"
	if musicPlayer != nil {
		c := musicPlayer.audioPlayer.Current()

		// Current Time
		m := (c / time.Minute) % 100
		s := (c / time.Second) % 60
		currentTimeStr = fmt.Sprintf("%02d:%02d", m, s)

		// Bar
		cw, ch := playerCurrentImage.Size()
		cx := int(time.Duration(w)*c/musicPlayer.total) + x - cw/2
		cy := y - (ch-h)/2
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(cx), float64(cy))
		screen.DrawImage(playerCurrentImage, op)
	}

	msg := fmt.Sprintf(`FPS: %0.2f
Press S to toggle Play/Pause
Press P to play SE
Press Z or X to change volume of the music
%s`, ebiten.CurrentFPS(), currentTimeStr)
	if musicPlayer == nil {
		msg += "\nNow Loading..."
	}
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	wavF, err := ebitenutil.OpenFile("_resources/audio/jab.wav")
	if err != nil {
		log.Fatal(err)
	}
	oggF, err := ebitenutil.OpenFile("_resources/audio/ragtime.ogg")
	if err != nil {
		log.Fatal(err)
	}
	audioContext = audio.NewContext(22050)
	go func() {
		s, err := wav.Decode(audioContext, wavF)
		if err != nil {
			log.Fatal(err)
			return
		}
		seCh <- s
		close(seCh)
	}()
	go func() {
		s, err := vorbis.Decode(audioContext, oggF)
		if err != nil {
			log.Fatal(err)
			return
		}
		p, err := audioContext.NewPlayer(s)
		if err != nil {
			log.Fatal(err)
			return
		}
		musicCh <- &Player{
			audioPlayer: p,
			total:       s.Len(),
		}
		close(musicCh)
		// TODO: Is this goroutine-safe?
		p.Play()
	}()
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Audio (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
