// Copyright 2015 Hajime Hoshi
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
	"log"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	sampleRate   = 48000
)

const (
	freqA  = 440.0
	freqAS = 466.2
	freqB  = 493.9
	freqC  = 523.3
	freqCS = 554.4
	freqD  = 587.3
	freqDS = 622.3
	freqE  = 659.3
	freqF  = 698.5
	freqFS = 740.0
	freqG  = 784.0
	freqGS = 830.6
)

// Twinkle, Twinkle, Little Star
var score = strings.Replace(
	`CCGGAAGR FFEEDDCR GGFFEEDR GGFFEEDR CCGGAAGR FFEEDDCR`,
	" ", "", -1)

// square fills out with square wave values with the specified volume, frequency and sequence.
func square(out []int16, volume float64, freq float64, sequence float64) {
	if freq == 0 {
		for i := 0; i < len(out); i++ {
			out[i] = 0
		}
		return
	}
	length := int(float64(sampleRate) / freq)
	if length == 0 {
		panic("invalid freq")
	}
	for i := 0; i < len(out); i++ {
		a := int16(volume * math.MaxInt16)
		if i%length < int(float64(length)*sequence) {
			a = -a
		}
		out[i] = a
	}
}

// toBytes returns the 2ch little endian 16bit byte sequence with the given left/right sequence.
func toBytes(l, r []int16) []byte {
	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	b := make([]byte, len(l)*4)
	for i := range l {
		b[4*i] = byte(l[i])
		b[4*i+1] = byte(l[i] >> 8)
		b[4*i+2] = byte(r[i])
		b[4*i+3] = byte(r[i] >> 8)
	}
	return b
}

type Game struct {
	scoreIndex  int
	frames      int
	currentNote rune

	audioContext *audio.Context
}

func NewGame() *Game {
	return &Game{
		audioContext: audio.NewContext(sampleRate),
	}
}

// playNote plays the note at scoreIndex of the score.
func (g *Game) playNote(scoreIndex int) rune {
	note := score[scoreIndex]

	// If the note is 'rest', play nothing.
	if note == 'R' {
		return rune(note)
	}

	freqs := []float64{freqC, freqD, freqE, freqF, freqG, freqA * 2, freqB * 2}
	freq := 0.0
	switch {
	case 'A' <= note && note <= 'B':
		freq = freqs[int(note)+len(freqs)-int('C')]
	case 'C' <= note && note <= 'G':
		freq = freqs[note-'C']
	default:
		panic("note out of range")
	}

	const vol = 1.0 / 16.0
	size := (ebiten.TPS()/2 - 2) * sampleRate / ebiten.TPS()
	l := make([]int16, size)
	r := make([]int16, size)
	square(l, vol, freq, 0.25)
	square(r, vol, freq, 0.25)

	p := g.audioContext.NewPlayerFromBytes(toBytes(l, r))
	p.Play()

	return rune(note)
}

func (g *Game) Update() error {
	// Play notes for each half second.
	if g.frames%30 == 0 && g.audioContext.IsReady() {
		g.currentNote = g.playNote(g.scoreIndex)
		g.scoreIndex++
		g.scoreIndex %= len(score)
	}
	g.frames++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	msg := "Note: "
	if g.currentNote == 'R' || g.currentNote == 0 {
		msg += "-"
	} else {
		msg += string(g.currentNote)
	}
	if !g.audioContext.IsReady() {
		msg += "\n\n(If the audio doesn't start,\n click the screen or press keys)"
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("PCM (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
