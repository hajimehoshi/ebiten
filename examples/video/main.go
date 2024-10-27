// Copyright 2024 The Ebitengine Authors
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
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// mpgURL is a URL of an example MPEG-1 video. The license is the following:
//
// https://commons.wikimedia.org/wiki/File:Shibuya_Crossing,_Tokyo,_Japan_(video).webm
// "Shibuya Crossing, Tokyo, Japan (video).webm" by Basile Morin
// The Creative Commons Attribution-Share Alike 4.0 International license
const mpgURL = "https://example-resources.ebitengine.org/shibuya.mpg"

type Game struct {
	player *mpegPlayer
	err    error
}

func (g *Game) Update() error {
	if g.err != nil {
		return g.err
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.err != nil {
		return
	}
	if err := g.player.Draw(screen); err != nil {
		g.err = err
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	// Initialize audio context.
	_ = audio.NewContext(48000)

	// If you want to play your own video, the video must be an MPEG-1 video with 48000 audio sample rate.
	// You can convert the video to MPEG-1 with the below command:
	//
	//     ffmpeg -i YOUR_VIDEO -c:v mpeg1video -q:v 8 -c:a mp2 -format mpeg -ar 48000 output.mpg
	//
	// You can adjust quality by changing -q:v value. A lower value indicates better quality.
	var in io.ReadCloser
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		in = f
	} else {
		res, err := http.Get(mpgURL)
		if err != nil {
			log.Fatal(err)
		}
		in = res.Body
		fmt.Println("Play the default video. You can specify a video file as an argument.")
	}

	player, err := newMPEGPlayer(in)
	if err != nil {
		log.Fatal(err)
	}
	g := &Game{
		player: player,
	}
	player.Play()

	ebiten.SetWindowTitle("Video (Ebitengine Demo)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
