// Copyright 2015 Hajime Hoshi
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

package mapeditor

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image/color"
	_ "image/png"
)

type MapTool int

const (
	MapToolScroll MapTool = iota
	MapToolPen
	MapToolFill
)

var mapToolsImage *ebiten.Image

func init() {
	var err error
	mapToolsImage, _, err = ebitenutil.NewImageFromFile("images/mapeditor/maptools.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
}

type MapTools struct {
	focused bool
	current MapTool
}

func NewMapTools() *MapTools {
	return &MapTools{}
}

func (m *MapTools) Update() {
}

func (m *MapTools) Current() MapTool {
	return m.current
}

func (m *MapTools) Draw(i *ebiten.Image, x, y, width, height int) error {
	i.DrawFilledRect(x+int(m.current)*32, y, 32, 32, color.White)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.GeoM.Scale(2, 2)
	if err := i.DrawImage(mapToolsImage, op); err != nil {
		return err
	}
	return nil
}
