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

// This is an example to implement an audio player.
// See examples/wav for a simpler example to play a sound file.

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	raudio "github.com/hajimehoshi/ebiten/v2/examples/resources/audio"
	riaudio "github.com/hajimehoshi/ebiten/v2/examples/resources/images/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480

	sampleRate = 48000
)

var (
	playerBarColor     = color.RGBA{0x80, 0x80, 0x80, 0xff}
	playerCurrentColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

var (
	playButtonImage  *ebiten.Image
	pauseButtonImage *ebiten.Image
	alertButtonImage *ebiten.Image
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(riaudio.Play_png))
	if err != nil {
		panic(err)
	}
	playButtonImage = ebiten.NewImageFromImage(img)

	img, _, err = image.Decode(bytes.NewReader(riaudio.Pause_png))
	if err != nil {
		panic(err)
	}
	pauseButtonImage = ebiten.NewImageFromImage(img)

	img, _, err = image.Decode(bytes.NewReader(riaudio.Alert_png))
	if err != nil {
		panic(err)
	}
	alertButtonImage = ebiten.NewImageFromImage(img)
}

type musicType int

const (
	typeOgg musicType = iota
	typeMP3
)

func (t musicType) String() string {
	switch t {
	case typeOgg:
		return "Ogg"
	case typeMP3:
		return "MP3"
	default:
		panic("not reached")
	}
}

// Player represents the current audio state.
type Player struct {
	game         *Game
	audioContext *audio.Context
	audioPlayer  *audio.Player
	current      time.Duration
	total        time.Duration
	seBytes      []byte
	seCh         chan []byte
	volume128    int
	musicType    musicType

	playButtonPosition  image.Point
	alertButtonPosition image.Point
}

func playerBarRect() (x, y, w, h int) {
	w, h = 600, 8
	x = (screenWidth - w) / 2
	y = screenHeight - h - 16
	return
}

func NewPlayer(game *Game, audioContext *audio.Context, musicType musicType) (*Player, error) {
	type audioStream interface {
		io.ReadSeeker
		Length() int64
	}

	const bytesPerSample = 4 // TODO: This should be defined in audio package

	var s audioStream

	switch musicType {
	case typeOgg:
		var err error
		s, err = vorbis.DecodeWithoutResampling(bytes.NewReader(raudio.Ragtime_ogg))
		if err != nil {
			return nil, err
		}
	case typeMP3:
		var err error
		s, err = mp3.DecodeWithoutResampling(bytes.NewReader(raudio.Ragtime_mp3))
		if err != nil {
			return nil, err
		}
	default:
		panic("not reached")
	}
	p, err := audioContext.NewPlayer(s)
	if err != nil {
		return nil, err
	}
	player := &Player{
		game:         game,
		audioContext: audioContext,
		audioPlayer:  p,
		total:        time.Second * time.Duration(s.Length()) / bytesPerSample / sampleRate,
		volume128:    128,
		seCh:         make(chan []byte),
		musicType:    musicType,
	}
	if player.total == 0 {
		player.total = 1
	}

	const buttonPadding = 16
	w := playButtonImage.Bounds().Dx()
	player.playButtonPosition.X = (screenWidth - w*2 + buttonPadding*1) / 2
	player.playButtonPosition.Y = screenHeight - 160

	player.alertButtonPosition.X = player.playButtonPosition.X + w + buttonPadding
	player.alertButtonPosition.Y = player.playButtonPosition.Y

	player.audioPlayer.Play()
	go func() {
		s, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(raudio.Jab_wav))
		if err != nil {
			log.Fatal(err)
			return
		}
		b, err := io.ReadAll(s)
		if err != nil {
			log.Fatal(err)
			return
		}
		player.seCh <- b
	}()
	return player, nil
}

func (p *Player) Close() error {
	return p.audioPlayer.Close()
}

func (p *Player) update() error {
	select {
	case p.seBytes = <-p.seCh:
		close(p.seCh)
		p.seCh = nil
	default:
	}

	if p.audioPlayer.IsPlaying() {
		p.current = p.audioPlayer.Position()
	}
	if err := p.seekBarIfNeeded(); err != nil {
		return err
	}
	p.switchPlayStateIfNeeded()
	p.playSEIfNeeded()
	p.updateVolumeIfNeeded()

	if inpututil.IsKeyJustPressed(ebiten.KeyU) {
		b := ebiten.IsRunnableOnUnfocused()
		ebiten.SetRunnableOnUnfocused(!b)
	}
	return nil
}

func (p *Player) shouldPlaySE() bool {
	if p.seBytes == nil {
		// Bytes for the SE is not loaded yet.
		return false
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		return true
	}
	r := image.Rectangle{
		Min: p.alertButtonPosition,
		Max: p.alertButtonPosition.Add(alertButtonImage.Bounds().Size()),
	}
	if image.Pt(ebiten.CursorPosition()).In(r) {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return true
		}
	}
	for _, id := range p.game.justPressedTouchIDs {
		if image.Pt(ebiten.TouchPosition(id)).In(r) {
			return true
		}
	}
	return false
}

func (p *Player) playSEIfNeeded() {
	if !p.shouldPlaySE() {
		return
	}
	sePlayer := p.audioContext.NewPlayerFromBytes(p.seBytes)
	sePlayer.Play()
}

func (p *Player) updateVolumeIfNeeded() {
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

func (p *Player) shouldSwitchPlayStateIfNeeded() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		return true
	}
	r := image.Rectangle{
		Min: p.playButtonPosition,
		Max: p.playButtonPosition.Add(playButtonImage.Bounds().Size()),
	}
	if image.Pt(ebiten.CursorPosition()).In(r) {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return true
		}
	}
	for _, id := range p.game.justPressedTouchIDs {
		if image.Pt(ebiten.TouchPosition(id)).In(r) {
			return true
		}
	}
	return false
}

func (p *Player) switchPlayStateIfNeeded() {
	if !p.shouldSwitchPlayStateIfNeeded() {
		return
	}
	if p.audioPlayer.IsPlaying() {
		p.audioPlayer.Pause()
		return
	}
	p.audioPlayer.Play()
}

func (p *Player) justPressedPosition() (int, int, bool) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return x, y, true
	}

	if len(p.game.justPressedTouchIDs) > 0 {
		x, y := ebiten.TouchPosition(p.game.justPressedTouchIDs[0])
		return x, y, true
	}

	return 0, 0, false
}

func (p *Player) seekBarIfNeeded() error {
	// Calculate the next seeking position from the current cursor position.
	x, y, ok := p.justPressedPosition()
	if !ok {
		return nil
	}
	bx, by, bw, bh := playerBarRect()
	const padding = 4
	if y < by-padding || by+bh+padding <= y {
		return nil
	}
	if x < bx || bx+bw <= x {
		return nil
	}
	pos := time.Duration(x-bx) * p.total / time.Duration(bw)
	p.current = pos
	if err := p.audioPlayer.SetPosition(pos); err != nil {
		return err
	}
	return nil
}

func (p *Player) draw(screen *ebiten.Image) {
	// Draw the bar.
	x, y, w, h := playerBarRect()
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), playerBarColor, true)

	// Draw the cursor on the bar.
	c := p.current
	cx := float32(x) + float32(w)*float32(p.current)/float32(p.total)
	cy := float32(y) + float32(h)/2
	vector.DrawFilledCircle(screen, cx, cy, 12, playerCurrentColor, true)

	// Compose the current time text.
	m := (c / time.Minute) % 100
	s := (c / time.Second) % 60
	currentTimeStr := fmt.Sprintf("%02d:%02d", m, s)

	// Draw buttons
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.playButtonPosition.X), float64(p.playButtonPosition.Y))
	if p.audioPlayer.IsPlaying() {
		screen.DrawImage(pauseButtonImage, op)
	} else {
		screen.DrawImage(playButtonImage, op)
	}
	op.GeoM.Reset()
	op.GeoM.Translate(float64(p.alertButtonPosition.X), float64(p.alertButtonPosition.Y))
	screen.DrawImage(alertButtonImage, op)

	// Draw the debug message.
	msg := fmt.Sprintf(`TPS: %0.2f
Press S to toggle Play/Pause
Press P to play SE
Press Z or X to change volume of the music
Press U to switch the runnable-on-unfocused state
Press A to switch Ogg and MP3 (Current: %s)
Current Time: %s
Current Volume: %d/128
Type: %s`, ebiten.ActualTPS(), p.musicType,
		currentTimeStr, int(p.audioPlayer.Volume()*128), p.musicType)
	ebitenutil.DebugPrint(screen, msg)
}

type Game struct {
	musicPlayer   *Player
	musicPlayerCh chan *Player
	errCh         chan error

	justPressedTouchIDs []ebiten.TouchID
}

func NewGame() (*Game, error) {
	audioContext := audio.NewContext(sampleRate)

	g := &Game{
		musicPlayerCh: make(chan *Player),
		errCh:         make(chan error),
	}

	m, err := NewPlayer(g, audioContext, typeOgg)
	if err != nil {
		return nil, err
	}

	g.musicPlayer = m
	return g, nil
}

func (g *Game) Update() error {
	select {
	case p := <-g.musicPlayerCh:
		g.musicPlayer = p
	case err := <-g.errCh:
		return err
	default:
	}

	g.justPressedTouchIDs = inpututil.AppendJustPressedTouchIDs(g.justPressedTouchIDs[:0])

	if g.musicPlayer != nil && inpututil.IsKeyJustPressed(ebiten.KeyA) {
		var t musicType
		switch g.musicPlayer.musicType {
		case typeOgg:
			t = typeMP3
		case typeMP3:
			t = typeOgg
		default:
			panic("not reached")
		}

		if err := g.musicPlayer.Close(); err != nil {
			return err
		}
		g.musicPlayer = nil

		go func() {
			p, err := NewPlayer(g, audio.CurrentContext(), t)
			if err != nil {
				g.errCh <- err
				return
			}
			g.musicPlayerCh <- p
		}()
	}

	if g.musicPlayer != nil {
		if err := g.musicPlayer.update(); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.musicPlayer != nil {
		g.musicPlayer.draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Audio (Ebitengine Demo)")
	g, err := NewGame()
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
