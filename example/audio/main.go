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
	"math/rand"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var frames = 0

const (
	hzA  = 440.0
	hzAS = 466.2
	hzB  = 493.9
	hzC  = 523.3
	hzCS = 554.4
	hzD  = 587.3
	hzDS = 622.3
	hzE  = 659.3
	hzF  = 698.5
	hzFS = 740.0
	hzG  = 784.0
	hzGS = 830.6
)

// TODO: Need API to get sample rate?
const sampleRate = 44100

func square(out []float32, hz float64, sequence float64) {
	length := int(sampleRate / hz)
	if length == 0 {
		panic("invalid hz")
	}
	for i := 0; i < len(out); i++ {
		a := float32(1.0)
		if i%length < int(float64(length)*sequence) {
			a = 0
		}
		out[i] = a
	}
}

func update(screen *ebiten.Image) error {
	defer func() {
		frames++
	}()

	const size = sampleRate / 60 // 3600 BPM
	notes := []float64{hzA, hzB, hzC, hzD, hzE, hzF, hzG, hzA * 2}
	if frames%30 == 0 {
		l := make([]float32, size*30)
		r := make([]float32, size*30)
		note := notes[rand.Intn(len(notes))]
		square(l, note, 0.5)
		square(r, note, 0.5)
		ebiten.AppendAudioBuffer(l, r)
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%0.2f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Rotate (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
