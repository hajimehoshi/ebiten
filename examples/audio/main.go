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
	"github.com/hajimehoshi/ebiten/audio/mp3"
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

type Input struct {
	mouseButtonStates map[ebiten.MouseButton]int
	keyStates         map[ebiten.Key]int
}

func (i *Input) update() {
	for _, key := range []ebiten.Key{ebiten.KeyP, ebiten.KeyS, ebiten.KeyX, ebiten.KeyZ} {
		if !ebiten.IsKeyPressed(key) {
			i.keyStates[key] = 0
		} else {
			i.keyStates[key]++
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		i.mouseButtonStates[ebiten.MouseButtonLeft] = 0
	} else {
		i.mouseButtonStates[ebiten.MouseButtonLeft]++
	}
}

func (i *Input) isKeyTriggered(key ebiten.Key) bool {
	return i.keyStates[key] == 1
}

func (i *Input) isKeyPressed(key ebiten.Key) bool {
	return i.keyStates[key] > 0
}

func (i *Input) isMouseButtonTriggered(mouseButton ebiten.MouseButton) bool {
	return i.mouseButtonStates[mouseButton] == 1
}

type Player struct {
	input        *Input
	audioContext *audio.Context
	audioPlayer  *audio.Player
	total        time.Duration
	seekedCh     chan error
	seBytes      []uint8
	seCh         chan []uint8
	volume128    int
	previousPos  time.Duration
}

var (
	musicPlayer *Player
)

func playerBarRect() (x, y, w, h int) {
	w, h = playerBarImage.Size()
	x = (screenWidth - w) / 2
	y = screenHeight - h - 16
	return
}

func NewPlayer(audioContext *audio.Context) (*Player, error) {
	const bytesPerSample = 4 // TODO: This should be defined in audio package
	wavF, err := ebitenutil.OpenFile("_resources/audio/jab.wav")
	if err != nil {
		return nil, err
	}
	mp3F, err := ebitenutil.OpenFile("_resources/audio/game2.mp3")
	if err != nil {
		return nil, err
	}
	s, err := mp3.Decode(audioContext, mp3F)
	if err != nil {
		return nil, err
	}
	p, err := audio.NewPlayer(audioContext, s)
	if err != nil {
		return nil, err
	}
	player := &Player{
		input: &Input{
			mouseButtonStates: map[ebiten.MouseButton]int{},
			keyStates:         map[ebiten.Key]int{},
		},
		audioContext: audioContext,
		audioPlayer:  p,
		total:        time.Second * time.Duration(s.Size()) / bytesPerSample / sampleRate,
		volume128:    128,
		seCh:         make(chan []uint8),
	}
	if player.total == 0 {
		player.total = 1
	}
	player.audioPlayer.Play()
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
		player.seCh <- b
	}()
	return player, nil
}

func (p *Player) update() error {
	p.input.update()
	select {
	case p.seBytes = <-p.seCh:
		close(p.seCh)
		p.seCh = nil
	case err := <-p.seekedCh:
		if err != nil {
			return err
		}
		close(p.seekedCh)
		p.seekedCh = nil
	default:
	}
	p.updateBar()
	p.updatePlayPause()
	p.updateSE()
	p.updateVolume()
	if err := p.audioContext.Update(); err != nil {
		return err
	}
	return nil
}

func (p *Player) updateSE() {
	if p.seBytes == nil {
		return
	}
	if !p.input.isKeyTriggered(ebiten.KeyP) {
		return
	}
	sePlayer, _ := audio.NewPlayerFromBytes(p.audioContext, p.seBytes)
	sePlayer.Play()
}

func (p *Player) updateVolume() {
	if p.input.isKeyPressed(ebiten.KeyZ) {
		p.volume128--
	}
	if p.input.isKeyPressed(ebiten.KeyX) {
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
	if !p.input.isKeyTriggered(ebiten.KeyS) {
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
	if !p.input.isMouseButtonTriggered(ebiten.MouseButtonLeft) {
		return
	}
	// Start seeking.
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
	prev := p.previousPos
	p.previousPos = c

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
	if p.audioPlayer.IsPlaying() && prev == c {
		msg += "\nLoading..."
	}
	ebitenutil.DebugPrint(screen, msg)
}

func update(screen *ebiten.Image) error {
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
	audioContext, err := audio.NewContext(sampleRate)
	if err != nil {
		log.Fatal(err)
	}
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
