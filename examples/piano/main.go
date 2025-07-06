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
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	arcadeFontSize = 8
)

var (
	arcadeFaceSource *text.GoTextFaceSource
)

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.PressStart2P_ttf))
	if err != nil {
		log.Fatal(err)
	}
	arcadeFaceSource = s
}

const (
	screenWidth  = 320
	screenHeight = 240
	sampleRate   = 48000
	baseFreq     = 220
)

// pianoAt returns an i-th sample of piano with the given frequency.
func pianoAt(i int, freq float32) float32 {
	// Create piano-like waves with multiple sin waves.
	amp := []float32{1.0, 0.8, 0.6, 0.4, 0.2}
	x := []float32{4.0, 2.0, 1.0, 0.5, 0.25}
	var v float32
	for j := 0; j < len(amp); j++ {
		// Decay
		a := amp[j] * float32(math.Exp(float64(-5*float32(i)*freq/baseFreq/(x[j]*sampleRate))))
		v += a * float32(math.Sin(2.0*math.Pi*float64(i)*float64(freq)*float64(j+1)/sampleRate))
	}
	return v / 5.0
}

// toBytes returns the 2ch little endian 16bit byte sequence with the given left/right sequence.
func toBytes(l, r []float32) []byte {
	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	b := make([]byte, len(l)*8)
	for i := range l {
		lv := math.Float32bits(l[i])
		rv := math.Float32bits(r[i])
		b[8*i] = byte(lv)
		b[8*i+1] = byte(lv >> 8)
		b[8*i+2] = byte(lv >> 16)
		b[8*i+3] = byte(lv >> 24)
		b[8*i+4] = byte(rv)
		b[8*i+5] = byte(rv >> 8)
		b[8*i+6] = byte(rv >> 16)
		b[8*i+7] = byte(rv >> 24)
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
		refData := make([]float32, length)
		for i := 0; i < length; i++ {
			refData[i] = pianoAt(i, refFreq)
		}

		for i := range keys {
			freq := baseFreq * math.Exp2(float64(i-1)/12.0)

			// Calculate the wave data for the freq.
			length := 4 * sampleRate * baseFreq / int(freq)
			l := make([]float32, length)
			r := make([]float32, length)
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
		vector.FillRect(pianoImage, float32(x), float32(y), float32(keyWidth-1), float32(height), color.White, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+keyWidth/2), float64(y+height-12))
		op.ColorScale.ScaleWithColor(color.Black)
		op.PrimaryAlign = text.AlignCenter
		text.Draw(pianoImage, k, &text.GoTextFace{
			Source: arcadeFaceSource,
			Size:   arcadeFontSize,
		}, op)
	}

	blackKeys := []string{"Q", "W", "", "R", "T", "", "U", "I", "O"}
	for i, k := range blackKeys {
		if k == "" {
			continue
		}
		x := i*keyWidth + 24
		height := 64
		vector.FillRect(pianoImage, float32(x), float32(y), float32(keyWidth-1), float32(height), color.Black, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+keyWidth/2), float64(y+height-12))
		op.ColorScale.ScaleWithColor(color.White)
		op.PrimaryAlign = text.AlignCenter
		text.Draw(pianoImage, k, &text.GoTextFace{
			Source: arcadeFaceSource,
			Size:   arcadeFontSize,
		}, op)
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
	audioContext *audio.Context
}

func NewGame() *Game {
	return &Game{
		audioContext: audio.NewContext(sampleRate),
	}
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
			g.playNote(baseFreq * float32(math.Exp2(float64(i-1)/12.0)))
		}
	}
	return nil
}

// playNote plays piano sound with the given frequency.
func (g *Game) playNote(freq float32) {
	f := int(freq)
	p := g.audioContext.NewPlayerF32FromBytes(pianoNoteSamples[f])
	p.Play()
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x80, 0x80, 0xc0, 0xff})
	screen.DrawImage(pianoImage, nil)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Piano (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
