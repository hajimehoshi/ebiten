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
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

var (
	text    = "Type on the keyboard:\n"
	counter = 0
)

func update(screen *ebiten.Image) error {
	text += string(ebiten.InputChars())
	ss := strings.Split(text, "\n")
	if len(ss) > 10 {
		text = strings.Join(ss[len(ss)-10:], "\n")
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}

	counter++

	if ebiten.IsRunningSlowly() {
		return nil
	}

	t := text
	if counter%60 < 30 {
		t += "_"
	}
	ebitenutil.DebugPrint(screen, t)
	return nil
}

func main() {
	if err := ebiten.Run(update, 320, 240, 2.0, "Runes (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
