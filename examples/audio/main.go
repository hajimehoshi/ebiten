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

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/exp/audio"
	"github.com/hajimehoshi/ebiten/exp/audio/vorbis"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	audioContext     *audio.Context
	audioLoadingDone chan struct{}
	audioLoaded      bool
)

func update(screen *ebiten.Image) error {
	audioContext.Update()
	if !audioLoaded {
		select {
		case <-audioLoadingDone:
			audioLoaded = true
		default:
		}
	}
	msg := fmt.Sprintf("FPS: %0.2f", ebiten.CurrentFPS())
	if !audioLoaded {
		msg += "\nNow Loading..."
	}
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	// Use a FLAC file so far: I couldn't find any good OGG/Vorbis decoder in pure Go.
	f, err := ebitenutil.OpenFile("_resources/audio/ragtime.ogg")
	if err != nil {
		log.Fatal(err)
	}
	// TODO: sampleRate should be obtained from the ogg file.
	audioContext = audio.NewContext(22050)
	audioLoadingDone = make(chan struct{})
	// TODO: This doesn't work synchronously on browsers because of decoding. Fix this.
	go func() {
		s, err := vorbis.Decode(audioContext, f)
		if err != nil {
			log.Fatal(err)
			return
		}
		p, err := audioContext.NewPlayer(s)
		if err != nil {
			log.Fatal(err)
			return
		}
		close(audioLoadingDone)
		p.Play()
	}()
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Audio (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
