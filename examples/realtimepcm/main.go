// Copyright 2022 The Ebiten Authors
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
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	sampleRate   = 48000
)

type Game struct {
	audioContext *audio.Context
	player       *audio.Player
	sineWave     *SineWave
}

type SineWave struct {
	frequency    int
	minFrequency int
	maxFrequency int

	// position is the position in the wave length in the range of [0, 1).
	position float64

	remaining []byte

	m sync.Mutex
}

func NewSineWave() *SineWave {
	return &SineWave{
		frequency:    440,
		minFrequency: 440,
		maxFrequency: 880,
	}
}

func (s *SineWave) Update(raisePitch bool) {
	s.m.Lock()
	defer s.m.Unlock()

	if raisePitch {
		if s.frequency < s.maxFrequency {
			s.frequency += 10
		}
	} else {
		if s.frequency > s.minFrequency {
			s.frequency -= 10
		}
	}
}

func (s *SineWave) Read(buf []byte) (int, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if len(s.remaining) > 0 {
		n := copy(buf, s.remaining)
		s.remaining = s.remaining[n:]
		return n, nil
	}

	var origBuf []byte
	if len(buf)%4 > 0 {
		origBuf = buf
		buf = make([]byte, len(origBuf)+4-len(origBuf)%4)
	}

	length := sampleRate / float64(s.frequency)
	p := int64(length * s.position)
	for i := 0; i < len(buf)/4; i++ {
		const max = 32767
		b := int16(math.Sin(2*math.Pi*float64(p)/float64(length)) * max)
		buf[4*i] = byte(b)
		buf[4*i+1] = byte(b >> 8)
		buf[4*i+2] = byte(b)
		buf[4*i+3] = byte(b >> 8)
		p++
	}

	s.position = float64(p) / float64(length)
	s.position = s.position - math.Floor(s.position)

	if origBuf != nil {
		n := copy(origBuf, buf)
		s.remaining = buf[n:]
		return n, nil
	}
	return len(buf), nil
}

func NewGame() *Game {
	return &Game{
		audioContext: audio.NewContext(sampleRate),
	}
}

func (g *Game) Update() error {
	if g.audioContext == nil {
		g.audioContext = audio.NewContext(sampleRate)
	}
	if g.player == nil {
		g.sineWave = NewSineWave()
		p, err := g.audioContext.NewPlayer(g.sineWave)
		if err != nil {
			return err
		}
		g.player = p
		g.player.Play()

		// Adjust the buffer size to reflect the audio source changes in real time.
		// Note that Ebitengine doesn't guarantee the audio quality when the buffer size is modified.
		// 1/20[s] should work in most cases, but this might cause glitches in some environments.
		g.player.SetBufferSize(time.Second / 20)
	}
	g.sineWave.Update(ebiten.IsKeyPressed(ebiten.KeyA))
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "This is an example of a real time PCM.\nPress and hold the A key.")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Real Time PCM (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
