// Copyright 2016 The Ebiten Authors
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

package twenty48

import (
	"errors"
	"image/color"
	"log"
	"math/rand"
	"sort"
	"strconv"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/text"
)

var (
	mplusSmallFont  font.Face
	mplusNormalFont font.Face
	mplusBigFont    font.Face
)

func init() {
	tt, err := truetype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusSmallFont = truetype.NewFace(tt, &truetype.Options{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	mplusNormalFont = truetype.NewFace(tt, &truetype.Options{
		Size:    32,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	mplusBigFont = truetype.NewFace(tt, &truetype.Options{
		Size:    48,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}

// TileData represents a tile information like a value and a position.
type TileData struct {
	value int
	x     int
	y     int
}

// Tile represents a tile information including TileData and animation states.
type Tile struct {
	current TileData

	// next represents a next tile information after moving.
	// next is empty when the tile is not about to move.
	next TileData

	movingCount       int
	startPoppingCount int
	poppingCount      int
}

// Pos returns the tile's current position.
// Pos is used only at testing so far.
func (t *Tile) Pos() (int, int) {
	return t.current.x, t.current.y
}

// NextPos returns the tile's next position.
// NextPos is used only at testing so far.
func (t *Tile) NextPos() (int, int) {
	return t.next.x, t.next.y
}

// Value returns the tile's current value.
// Value is used only at testing so far.
func (t *Tile) Value() int {
	return t.current.value
}

// NextValue returns the tile's current value.
// NextValue is used only at testing so far.
func (t *Tile) NextValue() int {
	return t.next.value
}

// NewTile creates a new Tile object.
func NewTile(value int, x, y int) *Tile {
	return &Tile{
		current: TileData{
			value: value,
			x:     x,
			y:     y,
		},
		startPoppingCount: maxPoppingCount,
	}
}

// IsMoving returns a boolean value indicating if the tile is animating.
func (t *Tile) IsMoving() bool {
	return 0 < t.movingCount
}

func (t *Tile) stopAnimation() {
	if 0 < t.movingCount {
		t.current = t.next
		t.next = TileData{}
	}
	t.movingCount = 0
	t.startPoppingCount = 0
	t.poppingCount = 0
}

func tileAt(tiles map[*Tile]struct{}, x, y int) *Tile {
	var result *Tile
	for t := range tiles {
		if t.current.x != x || t.current.y != y {
			continue
		}
		if result != nil {
			panic("not reach")
		}
		result = t
	}
	return result
}

func currentOrNextTileAt(tiles map[*Tile]struct{}, x, y int) *Tile {
	var result *Tile
	for t := range tiles {
		if 0 < t.movingCount {
			if t.next.x != x || t.next.y != y || t.next.value == 0 {
				continue
			}
		} else {
			if t.current.x != x || t.current.y != y {
				continue
			}
		}
		if result != nil {
			panic("not reach")
		}
		result = t
	}
	return result
}

const (
	maxMovingCount  = 5
	maxPoppingCount = 6
)

// MoveTiles moves tiles in the given tiles map if possible.
// MoveTiles returns true if there are tiles that are to move, otherwise false.
//
// When MoveTiles is called, all tiles must not be about to move.
func MoveTiles(tiles map[*Tile]struct{}, size int, dir Dir) bool {
	vx, vy := dir.Vector()
	tx := []int{}
	ty := []int{}
	for i := 0; i < size; i++ {
		tx = append(tx, i)
		ty = append(ty, i)
	}
	if vx > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(tx)))
	}
	if vy > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(ty)))
	}

	moved := false
	for _, j := range ty {
		for _, i := range tx {
			t := tileAt(tiles, i, j)
			if t == nil {
				continue
			}
			if t.next != (TileData{}) {
				panic("not reach")
			}
			if t.IsMoving() {
				panic("not reach")
			}
			// (ii, jj) is the next position for tile t.
			// (ii, jj) is updated until a mergeable tile is found or
			// the tile t can't be moved any more.
			ii := i
			jj := j
			for {
				ni := ii + vx
				nj := jj + vy
				if ni < 0 || ni >= size || nj < 0 || nj >= size {
					break
				}
				tt := currentOrNextTileAt(tiles, ni, nj)
				if tt == nil {
					ii = ni
					jj = nj
					moved = true
					continue
				}
				if t.current.value != tt.current.value {
					break
				}
				if 0 < tt.movingCount && tt.current.value != tt.next.value {
					// tt is already being merged with another tile.
					// Break here without updating (ii, jj).
					break
				}
				ii = ni
				jj = nj
				moved = true
				break
			}
			// next is the next state of the tile t.
			next := TileData{}
			next.value = t.current.value
			// If there is a tile at the next position (ii, jj), this should be
			// mergeable. Let's merge.
			if tt := currentOrNextTileAt(tiles, ii, jj); tt != t && tt != nil {
				next.value = t.current.value + tt.current.value
				tt.next.value = 0
				tt.next.x = ii
				tt.next.y = jj
				tt.movingCount = maxMovingCount
			}
			next.x = ii
			next.y = jj
			if t.current != next {
				t.next = next
				t.movingCount = maxMovingCount
			}
		}
	}
	if !moved {
		for t := range tiles {
			t.next = TileData{}
			t.movingCount = 0
		}
	}
	return moved
}

func addRandomTile(tiles map[*Tile]struct{}, size int) error {
	cells := make([]bool, size*size)
	for t := range tiles {
		if t.IsMoving() {
			panic("not reach")
		}
		i := t.current.x + t.current.y*size
		cells[i] = true
	}
	availableCells := []int{}
	for i, b := range cells {
		if b {
			continue
		}
		availableCells = append(availableCells, i)
	}
	if len(availableCells) == 0 {
		return errors.New("twenty48: there is no space to add a new tile")
	}
	c := availableCells[rand.Intn(len(availableCells))]
	v := 2
	if rand.Intn(10) == 0 {
		v = 4
	}
	x := c % size
	y := c / size
	t := NewTile(v, x, y)
	tiles[t] = struct{}{}
	return nil
}

// Update updates the tile's animation states.
func (t *Tile) Update() error {
	switch {
	case 0 < t.movingCount:
		t.movingCount--
		if t.movingCount == 0 {
			if t.current.value != t.next.value && 0 < t.next.value {
				t.poppingCount = maxPoppingCount
			}
			t.current = t.next
			t.next = TileData{}
		}
	case 0 < t.startPoppingCount:
		t.startPoppingCount--
	case 0 < t.poppingCount:
		t.poppingCount--
	}
	return nil
}

func colorToScale(clr color.Color) (float64, float64, float64, float64) {
	r, g, b, a := clr.RGBA()
	rf := float64(r) / 0xffff
	gf := float64(g) / 0xffff
	bf := float64(b) / 0xffff
	af := float64(a) / 0xffff
	// Convert to non-premultiplied alpha components.
	if 0 < af {
		rf /= af
		gf /= af
		bf /= af
	}
	return rf, gf, bf, af
}

func mean(a, b int, rate float64) int {
	return int(float64(a)*(1-rate) + float64(b)*rate)
}

func meanF(a, b float64, rate float64) float64 {
	return a*(1-rate) + b*rate
}

const (
	tileSize   = 80
	tileMargin = 4
)

var (
	tileImage *ebiten.Image
)

func init() {
	tileImage, _ = ebiten.NewImage(tileSize, tileSize, ebiten.FilterDefault)
	tileImage.Fill(color.White)
}

// Draw draws the current tile to the given boardImage.
func (t *Tile) Draw(boardImage *ebiten.Image) {
	i, j := t.current.x, t.current.y
	ni, nj := t.next.x, t.next.y
	v := t.current.value
	if v == 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	x := i*tileSize + (i+1)*tileMargin
	y := j*tileSize + (j+1)*tileMargin
	nx := ni*tileSize + (ni+1)*tileMargin
	ny := nj*tileSize + (nj+1)*tileMargin
	switch {
	case 0 < t.movingCount:
		rate := 1 - float64(t.movingCount)/maxMovingCount
		x = mean(x, nx, rate)
		y = mean(y, ny, rate)
	case 0 < t.startPoppingCount:
		rate := 1 - float64(t.startPoppingCount)/float64(maxPoppingCount)
		scale := meanF(0.0, 1.0, rate)
		op.GeoM.Translate(float64(-tileSize/2), float64(-tileSize/2))
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(tileSize/2), float64(tileSize/2))
	case 0 < t.poppingCount:
		const maxScale = 1.2
		rate := 0.0
		if maxPoppingCount*2/3 <= t.poppingCount {
			// 0 to 1
			rate = 1 - float64(t.poppingCount-2*maxPoppingCount/3)/float64(maxPoppingCount/3)
		} else {
			// 1 to 0
			rate = float64(t.poppingCount) / float64(maxPoppingCount*2/3)
		}
		scale := meanF(1.0, maxScale, rate)
		op.GeoM.Translate(float64(-tileSize/2), float64(-tileSize/2))
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(tileSize/2), float64(tileSize/2))
	}
	op.GeoM.Translate(float64(x), float64(y))
	r, g, b, a := colorToScale(tileBackgroundColor(v))
	op.ColorM.Scale(r, g, b, a)
	boardImage.DrawImage(tileImage, op)
	str := strconv.Itoa(v)

	f := mplusBigFont
	switch {
	case 3 < len(str):
		f = mplusSmallFont
	case 2 < len(str):
		f = mplusNormalFont
	}

	bound, _ := font.BoundString(f, str)
	w := (bound.Max.X - bound.Min.X).Ceil()
	h := (bound.Max.Y - bound.Min.Y).Ceil()
	x = x + (tileSize-w)/2
	y = y + (tileSize-h)/2 + h
	text.Draw(boardImage, str, f, x, y, tileColor(v))
}
