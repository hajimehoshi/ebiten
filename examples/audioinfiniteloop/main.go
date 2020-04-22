// Copyright 2017 The Ebiten Authors
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

// +build example jsgo

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/vorbis"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	raudio "github.com/hajimehoshi/ebiten/examples/resources/audio"
)

const (
	screenWidth  = 320
	screenHeight = 240
	sampleRate   = 22050

	introLengthInSecond = 5
	loopLengthInSecond  = 4
)

var audioContext *audio.Context

func init() {
	var err error
	audioContext, err = audio.NewContext(sampleRate)
	if err != nil {
		log.Fatal(err)
	}
}

type Game struct {
	player *audio.Player
}

func (g *Game) Update(screen *ebiten.Image) error {
	if g.player != nil {
		return nil
	}

	// Decode an Ogg file.
	// oggS is a decoded io.ReadCloser and io.Seeker.
	oggS, err := vorbis.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Ragtime_ogg))
	if err != nil {
		return err
	}

	// Create an infinite loop stream from the decoded bytes.
	// s is still an io.ReadCloser and io.Seeker.
	s := audio.NewInfiniteLoopWithIntro(oggS, introLengthInSecond*4*sampleRate, loopLengthInSecond*4*sampleRate)

	g.player, err = audio.NewPlayer(audioContext, s)
	if err != nil {
		return err
	}

	// Play the infinite-length stream. This never ends.
	g.player.Play()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	pos := g.player.Current()
	if pos > 5*time.Second {
		pos = (g.player.Current()-5*time.Second)%(4*time.Second) + 5*time.Second
	}
	msg := fmt.Sprintf(`TPS: %0.2f
This is an example using
audio.NewInfiniteLoopWithIntro.

Intro:   0[s] - %[2]d[s]
Loop:    %[2]d[s] - %[3]d[s]
Current: %0.2[4]f[s]`, ebiten.CurrentTPS(), introLengthInSecond, introLengthInSecond+loopLengthInSecond, float64(pos)/float64(time.Second))
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Audio Infinite Loop (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
