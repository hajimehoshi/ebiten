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

// +build example

package main

import (
	"errors"
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

type stream struct {
	position int64
}

func (s *stream) Read(data []byte) (int, error) {
	if len(data)%4 != 0 {
		return 0, errors.New("len(data) % 4 must be 0")
	}
	const length = sampleRate / frequency // TODO: This should be integer?
	p := s.position / 4
	for i := 0; i < len(data)/4; i++ {
		const max = (1<<15 - 1) / 2
		b := int16(math.Sin(2*math.Pi*float64(p)/length) * max)
		data[4*i] = byte(b)
		data[4*i+1] = byte(b >> 8)
		data[4*i+2] = byte(b)
		data[4*i+3] = byte(b >> 8)
		p++
	}
	s.position += int64(len(data))
	s.position %= length * 4
	return len(data), nil
}

func (s *stream) Seek(offset int64, whence int) (int64, error) {
	const length = sampleRate / frequency
	switch whence {
	case 0:
		s.position = offset
	case 1:
		s.position += offset
	case 2:
		return 0, errors.New("whence must be 0 or 1")
	}
	s.position %= length * 4
	return s.position, nil
}

func (s *stream) Close() error {
	return nil
}

var player *audio.Player

func update(screen *ebiten.Image) error {
	if player == nil {
		var err error
		player, err = audio.NewPlayer(audioContext, &stream{})
		if err != nil {
			return err
		}
		player.Play()
	}
	if ebiten.IsRunningSlowly() {
		return nil
	}
	msg := fmt.Sprintf("FPS: %0.2f\nThis is an example using infinite audio stream.", ebiten.CurrentFPS())
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Sine Wave (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
