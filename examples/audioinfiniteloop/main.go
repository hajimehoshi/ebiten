// Copyright 2017 The Ebiten Authors
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
	"log"
	"path/filepath"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/wav"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

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

var player *audio.Player

func update(screen *ebiten.Image) error {
	if player == nil {
		wavF, err := ebitenutil.OpenFile(filepath.Join("_resources", "audio", "jab.wav"))
		if err != nil {
			return err
		}

		wavS, err := wav.Decode(audioContext, wavF)
		if err != nil {
			return err
		}

		s := audio.NewInfiniteLoop(wavS, wavS.Size())

		player, err = audio.NewPlayer(audioContext, s)
		if err != nil {
			return err
		}
		player.Play()
	}

	if ebiten.IsRunningSlowly() {
		return nil
	}

	msg := fmt.Sprintf("FPS: %0.2f\nThis is an example using audio.NewInfiniteLoop.", ebiten.CurrentFPS())
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Audio Infinite Loop (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
