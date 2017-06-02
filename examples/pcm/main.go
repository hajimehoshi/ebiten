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

// +build example

package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
	sampleRate   = 44100
)

var audioContext *audio.Context

func init() {
	var err error
	audioContext, err = audio.NewContext(sampleRate)
	if err != nil {
		log.Fatal(err)
	}
}

var frames = 0

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
const score = `CCGGAAGR FFEEDDCR GGFFEEDR GGFFEEDR CCGGAAGR FFEEDDCR`

var scoreIndex = 0

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

func addNote() rune {
	size := sampleRate / ebiten.FPS
	notes := []float64{freqC, freqD, freqE, freqF, freqG, freqA * 2, freqB * 2}

	defer func() {
		scoreIndex++
		scoreIndex %= len(score)
	}()
	l := make([]int16, size*30)
	r := make([]int16, size*30)
	note := score[scoreIndex]
	for note == ' ' {
		scoreIndex++
		scoreIndex %= len(score)
		note = score[scoreIndex]
	}
	freq := 0.0
	switch {
	case note == 'R':
		freq = 0
	case note <= 'B':
		freq = notes[int(note)+len(notes)-int('C')]
	default:
		freq = notes[note-'C']
	}
	vol := 1.0 / 16.0
	square(l, vol, freq, 0.25)
	square(r, vol, freq, 0.25)
	b := toBytes(l, r)
	p, _ := audio.NewPlayerFromBytes(audioContext, b)
	p.Play()
	return rune(note)
}

var currentNote rune

func update(screen *ebiten.Image) error {
	if frames%30 == 0 {
		currentNote = addNote()
	}
	frames++
	if err := audioContext.Update(); err != nil {
		return err
	}
	if ebiten.IsRunningSlowly() {
		return nil
	}
	msg := "Note: "
	if currentNote == 'R' {
		msg += "-"
	} else {
		msg += string(currentNote)
	}
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "PCM (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
