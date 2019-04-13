// Copyright 2019 The Ebiten Authors
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
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

func update(screen *ebiten.Image) error {
	if ebiten.IsDrawingSkipped() {
		return nil
	}

	var path vector.Path
	path.MoveTo(20, 20)
	path.LineTo(30, 50)

	op := &vector.DrawPathOptions{}
	op.LineWidth = 4
	op.StrokeColor = color.White
	path.Draw(screen, op)

	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Vector (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
