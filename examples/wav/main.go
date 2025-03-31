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

package main

import (
	"bytes"
	"flag"
	"image"
	"io"
	"log"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	raudio "github.com/hajimehoshi/ebiten/v2/examples/resources/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	sampleRate   = 48000
)

var (
	flagBitsPerSample = flag.Int("bits", 16, "bits per sample")
)

type Game struct {
	debugui debugui.DebugUI

	audioContext *audio.Context
	audioPlayer  *audio.Player
}

func NewGame() (*Game, error) {
	g := &Game{}

	var err error
	// Initialize audio context.
	g.audioContext = audio.NewContext(sampleRate)

	// In this example, embedded resource "Jab_wav" is used.
	//
	// If you want to use a wav file, open this and pass the file stream to wav.Decode.
	// Note that file's Close() should not be closed here
	// since audio.Player manages stream state.
	//
	//     f, err := os.Open("jab.wav")
	//     if err != nil {
	//         return err
	//     }
	//
	//     d, err := wav.DecodeF32(f)
	//     ...

	// Decode wav-formatted data and retrieve decoded PCM stream.
	var r io.Reader
	switch *flagBitsPerSample {
	case 8:
		r = bytes.NewReader(raudio.Jab8_wav)
	default:
		r = bytes.NewReader(raudio.Jab_wav)
	}
	d, err := wav.DecodeF32(r)
	if err != nil {
		return nil, err
	}

	// Create an audio.Player that has one stream.
	g.audioPlayer, err = g.audioContext.NewPlayerF32(d)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (g *Game) Update() error {
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		var outErr error
		ctx.Window("WAV", image.Rect(10, 10, 210, 110), func(layout debugui.ContainerLayout) {
			if g.audioPlayer.IsPlaying() {
				ctx.Text("Bump!")
			} else {
				play := func() {
					// As audioPlayer has one stream and remembers the playing position,
					// rewinding is needed before playing when reusing audioPlayer.
					if err := g.audioPlayer.Rewind(); err != nil {
						outErr = err
						return
					}
					g.audioPlayer.Play()
				}
				ctx.Button("Play [P]").On(play)
				if inpututil.IsKeyJustPressed(ebiten.KeyP) {
					play()
				}
			}
		})
		return outErr
	}); err != nil {
		return err
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	flag.Parse()
	g, err := NewGame()
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("WAV (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
