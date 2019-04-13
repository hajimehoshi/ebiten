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

	// E
	path.MoveTo(20, 20)
	path.LineTo(20, 60)
	path.MoveTo(20, 20)
	path.LineTo(60, 20)
	path.MoveTo(20, 40)
	path.LineTo(60, 40)
	path.MoveTo(20, 60)
	path.LineTo(60, 60)

	// B
	path.MoveTo(80, 20)
	path.LineTo(80, 60)
	path.LineTo(110, 60)
	path.LineTo(120, 50)
	path.LineTo(110, 40)
	path.LineTo(120, 30)
	path.LineTo(110, 20)
	path.LineTo(80, 20)
	path.MoveTo(80, 40)
	path.LineTo(110, 40)

	// I
	path.MoveTo(140, 20)
	path.LineTo(140, 60)

	// T
	path.MoveTo(160, 20)
	path.LineTo(200, 20)
	path.MoveTo(180, 20)
	path.LineTo(180, 60)

	// E
	path.MoveTo(220, 20)
	path.LineTo(220, 60)
	path.MoveTo(220, 20)
	path.LineTo(260, 20)
	path.MoveTo(220, 40)
	path.LineTo(260, 40)
	path.MoveTo(220, 60)
	path.LineTo(260, 60)

	// N
	path.MoveTo(280, 60)
	path.LineTo(280, 20)
	path.LineTo(320, 60)
	path.LineTo(320, 20)

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
