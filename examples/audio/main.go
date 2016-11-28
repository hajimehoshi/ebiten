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

// +build example

package main

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/vorbis"
	"github.com/hajimehoshi/ebiten/audio/wav"
	"github.com/hajimehoshi/ebiten/ebitenutil"
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
	if err := playerBarImage.Fill(&color.RGBA{0x80, 0x80, 0x80, 0xff}); err != nil {
		log.Fatal(err)
	}

	playerCurrentImage, err = ebiten.NewImage(4, 10, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := playerCurrentImage.Fill(&color.RGBA{0xff, 0xff, 0xff, 0xff}); err != nil {
		log.Fatal(err)
	}
}

type Player struct {
	audioPlayer *audio.Player
	total       time.Duration
	seekedCh    chan error
}

var (
	audioContext     *audio.Context
	musicPlayer      *Player
	seBytes          []byte
	musicCh          = make(chan *Player)
	seCh             = make(chan []byte)
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
	if seBytes == nil {
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
	sePlayer, err := audio.NewPlayerFromBytes(audioContext, seBytes)
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
	if p.seekedCh != nil {
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
	const padding = 4
	if y < by-padding || by+bh+padding <= y {
		return nil
	}
	if x < bx || bx+bw <= x {
		return nil
	}
	pos := time.Duration(x-bx) * p.total / time.Duration(bw)
	p.seekedCh = make(chan error, 1)
	go func() {
		// This can't be done parallely! !?!?
		p.seekedCh <- p.audioPlayer.Seek(pos)
	}()
	return nil
}

func (p *Player) close() error {
	return p.audioPlayer.Close()
}

func update(screen *ebiten.Image) error {
	if musicPlayer == nil {
		select {
		case musicPlayer = <-musicCh:
		default:
		}
	}
	if seBytes == nil {
		select {
		case seBytes = <-seCh:
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
	if err := screen.DrawImage(playerBarImage, op); err != nil {
		return err
	}
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
		if err := screen.DrawImage(playerCurrentImage, op); err != nil {
			return err
		}
	}

	msg := fmt.Sprintf(`FPS: %0.2f
Press S to toggle Play/Pause
Press P to play SE
Press Z or X to change volume of the music
%s`, ebiten.CurrentFPS(), currentTimeStr)
	if musicPlayer == nil {
		msg += "\nNow Loading..."
	} else if musicPlayer.seekedCh != nil {
		select {
		case err := <-musicPlayer.seekedCh:
			if err != nil {
				return err
			}
			close(musicPlayer.seekedCh)
			musicPlayer.seekedCh = nil
		default:
			msg += "\nSeeking..."
		}
	}
	if err := ebitenutil.DebugPrint(screen, msg); err != nil {
		return err
	}
	if err := audioContext.Update(); err != nil {
		return err
	}
	return nil
}

func main() {
	wavF, err := ebitenutil.OpenFile("_resources/audio/jab.wav")
	if err != nil {
		log.Fatal(err)
	}
	oggF, err := ebitenutil.OpenFile("_resources/audio/game.ogg")
	if err != nil {
		log.Fatal(err)
	}
	const sampleRate = 44100
	const bytesPerSample = 4 // TODO: This should be defined in audio package
	audioContext, err = audio.NewContext(sampleRate)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		s, err := wav.Decode(audioContext, wavF)
		if err != nil {
			log.Fatal(err)
			return
		}
		b, err := ioutil.ReadAll(s)
		if err != nil {
			log.Fatal(err)
			return
		}
		seCh <- b
		close(seCh)
	}()
	go func() {
		s, err := vorbis.Decode(audioContext, oggF)
		if err != nil {
			log.Fatal(err)
			return
		}
		p, err := audio.NewPlayer(audioContext, s)
		if err != nil {
			log.Fatal(err)
			return
		}
		musicCh <- &Player{
			audioPlayer: p,
			total:       time.Second * time.Duration(s.Size()) / bytesPerSample / sampleRate,
		}
		close(musicCh)
		// TODO: Is this goroutine-safe?
		if err := p.Play(); err != nil {
			log.Fatal(err)
			return
		}
	}()
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Audio (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
	if musicPlayer != nil {
		if err := musicPlayer.close(); err != nil {
			log.Fatal(err)
		}
	}
}
