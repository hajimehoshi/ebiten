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

const (
	tileSetX      = 16
	tileSetY      = 16
	tileSetWidth  = TileSetXNum * TileWidth
	tileSetHeight = TileSetYNum * TileHeight
	mapViewX      = 16 + TileWidth*TileSetXNum + 16
	mapViewY      = 16
	mapViewWidth  = 720
	mapViewHeight = 720
)

type MainEditor struct {
	input        Input
	tileSetView  *TileSetView
	tileSetPanel *Panel
	mapView      *MapView
	mapPanel     *Panel
	mapCursorX   int
	mapCursorY   int
}

func NewMainEditor(tileSet *TileSet, m *Map) (*MainEditor, error) {
	tileSetView := NewTileSetView(tileSet)
	tileSetPanel, err := NewPanel(tileSetView, tileSetX, tileSetY, tileSetWidth, tileSetHeight)
	if err != nil {
		return nil, err
	}
	mapView := NewMapView(m)
	mapPanel, err := NewPanel(mapView, mapViewX, mapViewY, mapViewWidth, mapViewHeight)
	if err != nil {
		return nil, err
	}
	return &MainEditor{
		tileSetView:  tileSetView,
		tileSetPanel: tileSetPanel,
		mapView:      mapView,
		mapPanel:     mapPanel,
	}, nil
}

func (m *MainEditor) Update() error {
	m.input.Update()

	if err := m.tileSetView.Update(&m.input, tileSetX, tileSetY, TileWidth*TileSetXNum, TileHeight*TileSetYNum); err != nil {
		return err
	}
	tileSet := m.tileSetView.tileSet
	if err := m.mapView.Update(&m.input, mapViewX, mapViewY, mapViewWidth, mapViewHeight, tileSet, m.tileSetView.selectedTile); err != nil {
		return err
	}
	return nil
}

func (m *MainEditor) Draw(screen *ebiten.Image) error {
	if err := m.tileSetPanel.Draw(screen); err != nil {
		return err
	}
	if err := m.mapPanel.Draw(screen); err != nil {
		return err
	}
	return nil
}
