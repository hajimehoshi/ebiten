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
	"io/ioutil"
	"log"
	"math"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/text"
)

const (
	arcadeFontSize = 8
)

var (
	arcadeFont font.Face
)

func init() {
	f, err := ebitenutil.OpenFile(ebitenutil.JoinStringsIntoFilePath("_resources", "fonts", "arcade_n.ttf"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	tt, err := truetype.Parse(b)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	arcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    arcadeFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}

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

var pcm = make([]float64, 4*sampleRate)

const baseFreq = 220

func init() {
	amp := []float64{1.0, 0.8, 0.6, 0.4, 0.2}
	x := []float64{4.0, 2.0, 1.0, 0.5, 0.25}
	for i := 0; i < len(pcm); i++ {
		v := 0.0
		for j := 0; j < len(amp); j++ {
			a := amp[j] * math.Exp(-5*float64(i)/(x[j]*sampleRate))
			v += a * math.Sin(2.0*math.Pi*float64(i)*baseFreq*float64(j+1)/sampleRate)
		}
		pcm[i] = v / 5.0
	}
}

var (
	noteCache = map[int][]byte{}
)

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

func addNote(freq float64, vol float64) {
	// TODO: Call Close method of *audio.Player.
	// However, this works without Close because Close is automatically called when GC
	// collects a *audio.Player object.
	f := int(freq)
	if n, ok := noteCache[f]; ok {
		p, _ := audio.NewPlayerFromBytes(audioContext, n)
		p.Play()
		return
	}
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
	n := toBytes(l, r)
	noteCache[f] = n
	p, _ := audio.NewPlayerFromBytes(audioContext, n)
	p.Play()
	return
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

var (
	imagePiano *ebiten.Image
)

func init() {
	imagePiano, _ = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterNearest)
	whiteKeys := []string{"A", "S", "D", "F", "G", "H", "J", "K", "L"}
	width := 24
	y := 48
	for i, k := range whiteKeys {
		x := i*width + 36
		height := 112
		ebitenutil.DrawRect(imagePiano, float64(x), float64(y), float64(width-1), float64(height), color.White)
		text.Draw(imagePiano, k, arcadeFont, x+8, y+height-8, color.Black)
	}

	blackKeys := []string{"Q", "W", "", "R", "T", "", "U", "I", "O"}
	for i, k := range blackKeys {
		if k == "" {
			continue
		}
		x := i*width + 24
		height := 64
		ebitenutil.DrawRect(imagePiano, float64(x), float64(y), float64(width-1), float64(height), color.Black)
		text.Draw(imagePiano, k, arcadeFont, x+8, y+height-8, color.White)
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
	if ebiten.IsRunningSlowly() {
		return nil
	}
	screen.Fill(color.RGBA{0x80, 0x80, 0xc0, 0xff})
	screen.DrawImage(imagePiano, nil)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Piano (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
