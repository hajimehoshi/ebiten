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
	"fmt"
	"image/color"
	"log"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

var (
	arcadeFont font.Face
)

func init() {
	tt, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const (
		arcadeFontSize = 8
		dpi            = 72
	)
	arcadeFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    arcadeFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

const (
	screenWidth  = 320
	screenHeight = 240
	sampleRate   = 44100
	baseFreq     = 220
)

var audioContext = audio.NewContext(sampleRate)

// pianoAt returns an i-th sample of piano with the given frequency.
func pianoAt(i int, freq float64) float64 {
	// Create piano-like waves with multiple sin waves.
	amp := []float64{1.0, 0.8, 0.6, 0.4, 0.2}
	x := []float64{4.0, 2.0, 1.0, 0.5, 0.25}
	v := 0.0
	for j := 0; j < len(amp); j++ {
		// Decay
		a := amp[j] * math.Exp(-5*float64(i)*freq/baseFreq/(x[j]*sampleRate))
		v += a * math.Sin(2.0*math.Pi*float64(i)*freq*float64(j+1)/sampleRate)
	}
	return v / 5.0
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

var (
	pianoNoteSamples       = map[int][]byte{}
	pianoNoteSamplesInited = false
	pianoNoteSamplesInitCh = make(chan struct{})
)

func init() {
	// Initialize piano data.
	// This takes a little long time (especially on browsers),
	// so run this asynchronously and notice the progress.
	go func() {
		// Create a reference data and use this for other frequency.
		const refFreq = 110
		length := 4 * sampleRate * baseFreq / refFreq
		refData := make([]int16, length)
		for i := 0; i < length; i++ {
			refData[i] = int16(pianoAt(i, refFreq) * math.MaxInt16)
		}

		for i := range keys {
			freq := baseFreq * math.Exp2(float64(i-1)/12.0)

			// Clculate the wave data for the freq.
			length := 4 * sampleRate * baseFreq / int(freq)
			l := make([]int16, length)
			r := make([]int16, length)
			for i := 0; i < length; i++ {
				idx := int(float64(i) * freq / refFreq)
				if len(refData) <= idx {
					break
				}
				l[i] = refData[idx]
			}
			copy(r, l)
			n := toBytes(l, r)
			pianoNoteSamples[int(freq)] = n
		}
		close(pianoNoteSamplesInitCh)
	}()
}

// playNote plays piano sound with the given frequency.
func playNote(freq float64) {
	f := int(freq)
	p := audio.NewPlayerFromBytes(audioContext, pianoNoteSamples[f])
	p.Play()
}

var (
	pianoImage = ebiten.NewImage(screenWidth, screenHeight)
)

func init() {
	const (
		keyWidth = 24
		y        = 48
	)

	whiteKeys := []string{"A", "S", "D", "F", "G", "H", "J", "K", "L"}
	for i, k := range whiteKeys {
		x := i*keyWidth + 36
		height := 112
		ebitenutil.DrawRect(pianoImage, float64(x), float64(y), float64(keyWidth-1), float64(height), color.White)
		text.Draw(pianoImage, k, arcadeFont, x+8, y+height-8, color.Black)
	}

	blackKeys := []string{"Q", "W", "", "R", "T", "", "U", "I", "O"}
	for i, k := range blackKeys {
		if k == "" {
			continue
		}
		x := i*keyWidth + 24
		height := 64
		ebitenutil.DrawRect(pianoImage, float64(x), float64(y), float64(keyWidth-1), float64(height), color.Black)
		text.Draw(pianoImage, k, arcadeFont, x+8, y+height-8, color.White)
	}
}

var (
	keys = []ebiten.Key{
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
)

type Game struct {
}

func (g *Game) Update() error {
	// The piano data is still being initialized.
	// Get the progress if available.
	if !pianoNoteSamplesInited {
		select {
		case <-pianoNoteSamplesInitCh:
			pianoNoteSamplesInited = true
		default:
		}
	}

	if pianoNoteSamplesInited {
		for i, key := range keys {
			if !inpututil.IsKeyJustPressed(key) {
				continue
			}
			playNote(baseFreq * math.Exp2(float64(i-1)/12.0))
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x80, 0x80, 0xc0, 0xff})
	screen.DrawImage(pianoImage, nil)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.CurrentTPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Piano (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
