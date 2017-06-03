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

	// This sample rate doesn't match with wav/ogg's sample rate,
	// but decoders adjust them.
	sampleRate = 48000
)

var (
	playerBarImage     *ebiten.Image
	playerCurrentImage *ebiten.Image
)

func init() {
	playerBarImage, _ = ebiten.NewImage(300, 4, ebiten.FilterNearest)
	playerBarImage.Fill(&color.RGBA{0x80, 0x80, 0x80, 0xff})

	playerCurrentImage, _ = ebiten.NewImage(4, 10, ebiten.FilterNearest)
	playerCurrentImage.Fill(&color.RGBA{0xff, 0xff, 0xff, 0xff})
}

type Player struct {
	audioContext     *audio.Context
	audioPlayer      *audio.Player
	total            time.Duration
	seekedCh         chan error
	mouseButtonState map[ebiten.MouseButton]int
	keyState         map[ebiten.Key]int
	volume128        int
}

var (
	musicPlayer *Player
	seBytes     []byte
	seCh        = make(chan []byte)
)

func playerBarRect() (x, y, w, h int) {
	w, h = playerBarImage.Size()
	x = (screenWidth - w) / 2
	y = screenHeight - h - 16
	return
}

func NewPlayer(audioContext *audio.Context) (*Player, error) {
	const bytesPerSample = 4 // TODO: This should be defined in audio package
	oggF, err := ebitenutil.OpenFile("_resources/audio/game.ogg")
	if err != nil {
		return nil, err
	}
	s, err := vorbis.Decode(audioContext, oggF)
	if err != nil {
		return nil, err
	}
	p, err := audio.NewPlayer(audioContext, s)
	if err != nil {
		return nil, err
	}
	player := &Player{
		audioContext:     audioContext,
		audioPlayer:      p,
		total:            time.Second * time.Duration(s.Size()) / bytesPerSample / sampleRate,
		mouseButtonState: map[ebiten.MouseButton]int{},
		keyState:         map[ebiten.Key]int{},
		volume128:        128,
	}
	player.audioPlayer.Play()
	return player, nil
}

func (p *Player) update() error {
	p.updateBar()
	p.updatePlayPause()
	p.updateSE()
	p.updateVolume()
	if err := p.audioContext.Update(); err != nil {
		return err
	}
	select {
	case err := <-p.seekedCh:
		if err != nil {
			return err
		}
		close(p.seekedCh)
		p.seekedCh = nil
	default:
	}
	return nil
}

func (p *Player) updateSE() {
	if seBytes == nil {
		return
	}
	if !ebiten.IsKeyPressed(ebiten.KeyP) {
		p.keyState[ebiten.KeyP] = 0
		return
	}
	p.keyState[ebiten.KeyP]++
	if p.keyState[ebiten.KeyP] != 1 {
		return
	}
	sePlayer, _ := audio.NewPlayerFromBytes(p.audioContext, seBytes)
	sePlayer.Play()
}

func (p *Player) updateVolume() {
	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		p.volume128--
	}
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		p.volume128++
	}
	if p.volume128 < 0 {
		p.volume128 = 0
	}
	if 128 < p.volume128 {
		p.volume128 = 128
	}
	p.audioPlayer.SetVolume(float64(p.volume128) / 128)
}

func (p *Player) updatePlayPause() {
	if !ebiten.IsKeyPressed(ebiten.KeyS) {
		p.keyState[ebiten.KeyS] = 0
		return
	}
	p.keyState[ebiten.KeyS]++
	if p.keyState[ebiten.KeyS] != 1 {
		return
	}
	if p.audioPlayer.IsPlaying() {
		p.audioPlayer.Pause()
		return
	}
	p.audioPlayer.Play()
}

func (p *Player) updateBar() {
	if p.seekedCh != nil {
		return
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		p.mouseButtonState[ebiten.MouseButtonLeft] = 0
		return
	}
	p.mouseButtonState[ebiten.MouseButtonLeft]++
	if p.mouseButtonState[ebiten.MouseButtonLeft] != 1 {
		return
	}
	x, y := ebiten.CursorPosition()
	bx, by, bw, bh := playerBarRect()
	const padding = 4
	if y < by-padding || by+bh+padding <= y {
		return
	}
	if x < bx || bx+bw <= x {
		return
	}
	pos := time.Duration(x-bx) * p.total / time.Duration(bw)
	p.seekedCh = make(chan error, 1)
	go func() {
		// This can't be done parallely! !?!?
		p.seekedCh <- p.audioPlayer.Seek(pos)
	}()
}

func (p *Player) close() error {
	return p.audioPlayer.Close()
}

func (p *Player) draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	x, y, w, h := playerBarRect()
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(playerBarImage, op)
	currentTimeStr := "00:00"
	c := p.audioPlayer.Current()

	// Current Time
	m := (c / time.Minute) % 100
	s := (c / time.Second) % 60
	currentTimeStr = fmt.Sprintf("%02d:%02d", m, s)

	// Bar
	cw, ch := playerCurrentImage.Size()
	cx := int(time.Duration(w)*c/p.total) + x - cw/2
	cy := y - (ch-h)/2
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(cx), float64(cy))
	screen.DrawImage(playerCurrentImage, op)

	msg := fmt.Sprintf(`FPS: %0.2f
Press S to toggle Play/Pause
Press P to play SE
Press Z or X to change volume of the music
%s`, ebiten.CurrentFPS(), currentTimeStr)
	if p.seekedCh != nil {
		msg += "\nSeeking..."
	}
	ebitenutil.DebugPrint(screen, msg)
}

func update(screen *ebiten.Image) error {
	if seBytes == nil {
		select {
		case seBytes = <-seCh:
		default:
		}
	}
	if err := musicPlayer.update(); err != nil {
		return err
	}
	if ebiten.IsRunningSlowly() {
		return nil
	}
	musicPlayer.draw(screen)
	return nil
}

func main() {
	wavF, err := ebitenutil.OpenFile("_resources/audio/jab.wav")
	if err != nil {
		log.Fatal(err)
	}
	audioContext, err := audio.NewContext(sampleRate)
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
	musicPlayer, err = NewPlayer(audioContext)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Audio (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
	if musicPlayer != nil {
		if err := musicPlayer.close(); err != nil {
			log.Fatal(err)
		}
	}
}
