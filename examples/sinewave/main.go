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

// +build example jsgo

package main

import (
	"fmt"
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
	frequency    = 440
)

var audioContext *audio.Context

func init() {
	var err error
	audioContext, err = audio.NewContext(sampleRate)
	if err != nil {
		log.Fatal(err)
	}
}

// stream is an infinite stream of 440 Hz sine wave.
type stream struct {
	position  int64
	remaining []byte
}

// Read is io.Reader's Read.
//
// Read fills the data with sine wave samples.
func (s *stream) Read(buf []byte) (int, error) {
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

	const length = int64(sampleRate / frequency)
	p := s.position / 4
	for i := 0; i < len(buf)/4; i++ {
		const max = (1<<15 - 1) / 2
		b := int16(math.Sin(2*math.Pi*float64(p)/float64(length)) * max)
		buf[4*i] = byte(b)
		buf[4*i+1] = byte(b >> 8)
		buf[4*i+2] = byte(b)
		buf[4*i+3] = byte(b >> 8)
		p++
	}

	s.position += int64(len(buf))
	s.position %= length * 4

	if origBuf != nil {
		n := copy(origBuf, buf)
		s.remaining = buf[n:]
		return n, nil
	}
	return len(buf), nil
}

// Close is io.Closer's Close.
func (s *stream) Close() error {
	return nil
}

var player *audio.Player

func update(screen *ebiten.Image) error {
	if player == nil {
		// Pass the (infinite) stream to audio.NewPlayer.
		// After calling Play, the stream never ends as long as the player object lives.
		var err error
		player, err = audio.NewPlayer(audioContext, &stream{})
		if err != nil {
			return err
		}
		player.Play()
	}
	if ebiten.IsDrawingSkipped() {
		return nil
	}
	msg := fmt.Sprintf("TPS: %0.2f\nThis is an example using infinite audio stream.", ebiten.CurrentTPS())
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Sine Wave (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
