// Copyright 2018 The Ebiten Authors
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
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	ebitenImage *ebiten.Image
)

func update(screen *ebiten.Image) error {
	if ebiten.IsRunningSlowly() {
		return nil
	}

	ebitenutil.DebugPrint(screen, "Nearest Filter (default) VS Linear Filter")

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4, 4)
	op.GeoM.Translate(64, 64)
	// By default, nearest filter is used.
	screen.DrawImage(ebitenImage, op)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4, 4)
	op.GeoM.Translate(64, 64+240)
	// Specify linear filter.
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(ebitenImage, op)

	return nil
}

func main() {
	var err error
	// Specifying filter on NewImage(FromImage) is just for backward compatibility.
	// Now specifying filter at DrawImageOptions is recommended.
	// Specify FilterDefault here, that means to prefer filter specified at DrawImageOptions.
	ebitenImage, _, err = ebitenutil.NewImageFromFile("_resources/images/ebiten.png", ebiten.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Filter (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
