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
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"log"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

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

const score = `CCGGAAGR FFEEDDCR GGFFEEDR GGFFEEDR CCGGAAGR FFEEDDCR`

var scoreIndex = 0

func square(out []float32, volume float64, freq float64, sequence float64) {
	if freq == 0 {
		for i := 0; i < len(out); i++ {
			out[i] = 0
		}
		return
	}
	length := int(float64(ebiten.AudioSampleRate()) / freq)
	if length == 0 {
		panic("invalid freq")
	}
	for i := 0; i < len(out); i++ {
		a := float32(volume)
		if i%length < int(float64(length)*sequence) {
			a = 0
		}
		out[i] = a
	}
}

func addNote() {
	size := ebiten.AudioSampleRate() / 60
	notes := []float64{freqC, freqD, freqE, freqF, freqG, freqA * 2, freqB * 2}

	defer func() {
		scoreIndex++
		scoreIndex %= len(score)
	}()
	l := make([]float32, size*30)
	r := make([]float32, size*30)
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
	vol := 1.0 / 32.0
	square(l, vol, freq, 0.5)
	square(r, vol, freq, 0.5)
	ebiten.AppendToAudioBuffer(0, l, r)
}

func update(screen *ebiten.Image) error {
	defer func() {
		frames++
	}()
	if frames%30 == 0 {
		addNote()
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Rotate (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
