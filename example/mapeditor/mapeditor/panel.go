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
)

type drawer interface {
	Draw(target *ebiten.Image, x, y, width, height int) error
}

type Panel struct {
	drawer    drawer
	panelX    int
	panelY    int
	regionX   int
	regionY   int
	width     int
	height    int
	offscreen *ebiten.Image
}

func NewPanel(drawer drawer, x, y, width, height int) (*Panel, error) {
	offscreen, err := ebiten.NewImage(width, height, ebiten.FilterNearest)
	if err != nil {
		return nil, err
	}
	return &Panel{
		drawer:    drawer,
		panelX:    x,
		panelY:    y,
		width:     width,
		height:    height,
		offscreen: offscreen,
	}, nil
}

func (p *Panel) Draw(target *ebiten.Image) error {
	if err := p.drawer.Draw(p.offscreen, p.regionX, p.regionY, p.width, p.height); err != nil {
		return err
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.panelX), float64(p.panelY))
	return target.DrawImage(p.offscreen, op)
}
