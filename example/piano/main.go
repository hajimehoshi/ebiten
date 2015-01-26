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
	"github.com/hajimehoshi/ebiten/exp/audio"
	"log"
	"math"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var pcm = make([]float64, 4*audio.SampleRate())

const baseFreq = 220

func init() {
	s := float64(audio.SampleRate())
	amp := []float64{1.0, 0.8, 0.6, 0.4, 0.2}
	x := []float64{4.0, 2.0, 1.0, 0.5, 0.25}
	for i := 0; i < len(pcm); i++ {
		v := 0.0
		twoPiF := 2.0 * math.Pi * baseFreq
		for j := 0; j < len(amp); j++ {
			a := amp[j] * math.Exp(-5*float64(i)/(x[j]*s))
			v += a * math.Sin(float64(i)*twoPiF*float64(j+1)/s)
		}
		pcm[i] = v / 5.0
	}
}

func addNote(freq float64, vol float64) {
	f := int(freq)
	length := len(pcm) * baseFreq / f
	l := make([]int16, length)
	r := make([]int16, length)
	j := 0
	jj := 0
	for i := 0; i < len(l); i++ {
		p := pcm[j]
		l[i] = int16(p * vol * math.MaxInt16)
		r[i] = l[i]
		jj += f
		j = jj / baseFreq
	}
	audio.Play(-1, l, r)
}

var keys = []ebiten.Key{
	ebiten.KeyQ,
	ebiten.KeyA,
	ebiten.KeyW,
	ebiten.KeyS,
	ebiten.KeyD,
	ebiten.KeyR,
	ebiten.KeyF,
	ebiten.KeyT,
	ebiten.KeyG,
	ebiten.KeyH,
	ebiten.KeyU,
	ebiten.KeyJ,
	ebiten.KeyI,
	ebiten.KeyK,
	ebiten.KeyO,
	ebiten.KeyL,
}

var keyStates = map[ebiten.Key]int{}

func init() {
	for _, key := range keys {
		keyStates[key] = 0
	}
}

func updateInput() {
	for _, key := range keys {
		if !ebiten.IsKeyPressed(key) {
			keyStates[key] = 0
			continue
		}
		keyStates[key]++
	}
}

func update(screen *ebiten.Image) error {
	updateInput()
	for i, key := range keys {
		if keyStates[key] != 1 {
			continue
		}
		addNote(220*math.Exp2(float64(i-1)/12.0), 1.0)
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Piano (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
