// Copyright 2026 The Ebitengine Authors
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

//go:build ebitenginevm

// This is a guest that runs IME sessions through an exp/textinput Composer and paints its state, so
// the host can verify the text-input round trip from the composited frame: red = no activity yet,
// blue = the expected composition is showing, green = the expected commit was applied at the caret,
// cyan = the expected replacement commit was applied, and white = anything unexpected. It is
// launched by a host; see vmhost's textinput test.
package main

import (
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

// The session parameters and expected responses mirror the host test; keep the two in sync.
const (
	textBeforeCaret = "ab"
	textAfterCaret  = "cd"
	wantComposition = "こん"
	wantCommit      = "こんにちは"
	// wantReplacedBuffer results from a commit replacing part of the surrounding text.
	wantReplacedBuffer = "aXYd"
)

type game struct {
	composer    textinput.Composer
	composition string
	buffer      string
}

func (g *game) Update() error {
	_, err := g.composer.Update()
	return err
}

func (g *game) Draw(screen *ebiten.Image) {
	switch {
	case g.buffer == textBeforeCaret+wantCommit+textAfterCaret:
		screen.Fill(color.RGBA{G: 0xff, A: 0xff})
	case g.buffer == wantReplacedBuffer:
		screen.Fill(color.RGBA{G: 0xff, B: 0xff, A: 0xff})
	case g.buffer == "" && g.composition == wantComposition:
		screen.Fill(color.RGBA{B: 0xff, A: 0xff})
	case g.buffer == "" && g.composition == "":
		screen.Fill(color.RGBA{R: 0xff, A: 0xff})
	default:
		screen.Fill(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	}
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	var g game
	g.composer.OnNewSession = func() *textinput.SessionOptions {
		return &textinput.SessionOptions{
			CaretBounds:     image.Rect(4, 8, 6, 24),
			TextBeforeCaret: textBeforeCaret,
			TextAfterCaret:  textAfterCaret,
		}
	}
	g.composer.OnComposition = func(c *textinput.Composition) {
		g.composition = c.Text()
	}
	g.composer.OnCommit = func(c *textinput.Commit) {
		// Apply the commit as an editor would, so a wrong replacement range surfaces as an
		// unexpected buffer.
		before, after := c.SurroundingText()
		g.buffer = before + c.Text() + after
	}
	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}
