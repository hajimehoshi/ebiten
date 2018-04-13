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

// +build example jsgo

package main

import (
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
)

var (
	text    = "Type on the keyboard:\n"
	counter = 0
)

// repeatingKeyPressed return true when key is pressed considering the repeat state.
func repeatingKeyPressed(key ebiten.Key) bool {
	const (
		delay    = 30
		interval = 3
	)
	d := inpututil.KeyPressDuration(key)
	if d == 1 {
		return true
	}
	if d >= delay && (d-delay)%interval == 0 {
		return true
	}
	return false
}

func update(screen *ebiten.Image) error {
	// Add a string from InputChars, that returns string input by users.
	// Note that InputChars result changes every frame, so you need to call this
	// every frame.
	text += string(ebiten.InputChars())

	// Adjust the string to be at most 10 lines.
	ss := strings.Split(text, "\n")
	if len(ss) > 10 {
		text = strings.Join(ss[len(ss)-10:], "\n")
	}

	// If the enter key is pressed, add a line break.
	if repeatingKeyPressed(ebiten.KeyEnter) || repeatingKeyPressed(ebiten.KeyKPEnter) {
		text += "\n"
	}

	// If the backspace key is pressed, remove one character.
	if repeatingKeyPressed(ebiten.KeyBackspace) {
		if len(text) >= 1 {
			text = text[:len(text)-1]
		}
	}

	counter++

	if ebiten.IsRunningSlowly() {
		return nil
	}

	// Blink the cursor.
	t := text
	if counter%60 < 30 {
		t += "_"
	}
	ebitenutil.DebugPrint(screen, t)
	return nil
}

func main() {
	if err := ebiten.Run(update, 320, 240, 2.0, "Typewriter (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
