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

package main

import (
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	sampleRate   = 48000
	frequency    = 440
)

// stream is an infinite stream of 440 Hz sine wave.
type stream struct {
	pos int64
}

// Read is io.Reader's Read.
//
// Read fills the data with sine wave samples.
func (s *stream) Read(buf []byte) (int, error) {
	const bytesPerSample = 8

	n := len(buf) / bytesPerSample * bytesPerSample

	const length = sampleRate / frequency
	for i := 0; i < n/bytesPerSample; i++ {
		v := math.Float32bits(float32(math.Sin(2 * math.Pi * float64(s.pos/bytesPerSample+int64(i)) / length)))
		buf[8*i] = byte(v)
		buf[8*i+1] = byte(v >> 8)
		buf[8*i+2] = byte(v >> 16)
		buf[8*i+3] = byte(v >> 24)
		buf[8*i+4] = byte(v)
		buf[8*i+5] = byte(v >> 8)
		buf[8*i+6] = byte(v >> 16)
		buf[8*i+7] = byte(v >> 24)
	}

	s.pos += int64(n)
	s.pos %= length * bytesPerSample

	return n, nil
}

// Close is io.Closer's Close.
func (s *stream) Close() error {
	return nil
}

type Game struct {
	audioContext *audio.Context
	player       *audio.Player
}

func (g *Game) Update() error {
	if g.audioContext == nil {
		g.audioContext = audio.NewContext(sampleRate)
	}
	if g.player == nil {
		// Pass the (infinite) stream to NewPlayer.
		// After calling Play, the stream never ends as long as the player object lives.
		var err error
		g.player, err = g.audioContext.NewPlayerF32(&stream{})
		if err != nil {
			return err
		}
		g.player.Play()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	msg := fmt.Sprintf("TPS: %0.2f\nThis is an example using infinite audio stream.", ebiten.ActualTPS())
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Sine Wave (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
