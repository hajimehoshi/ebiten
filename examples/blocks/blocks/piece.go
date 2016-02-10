// Copyright 2014 Hajime Hoshi
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

package blocks

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

var imageBlocks *ebiten.Image

func init() {
	var err error
	imageBlocks, _, err = ebitenutil.NewImageFromFile("images/blocks/blocks.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
}

type Angle int

const (
	Angle0 Angle = iota
	Angle90
	Angle180
	Angle270
)

func (a Angle) RotateRight() Angle {
	if a == Angle270 {
		return Angle0
	}
	return a + 1
}

func (a Angle) RotateLeft() Angle {
	if a == Angle0 {
		return Angle270
	}
	return a - 1
}

type BlockType int

const (
	BlockTypeNone BlockType = iota
	BlockType1
	BlockType2
	BlockType3
	BlockType4
	BlockType5
	BlockType6
	BlockType7
	BlockTypeMax = BlockType7
)

type Piece struct {
	blockType BlockType
	blocks    [][]bool
}

func toBlocks(ints [][]int) [][]bool {
	blocks := make([][]bool, len(ints))
	for j, row := range ints {
		blocks[j] = make([]bool, len(row))
	}
	// Tranpose the argument matrix.
	for i, col := range ints {
		for j, v := range col {
			blocks[j][i] = v != 0
		}
	}
	return blocks
}

var Pieces = map[BlockType]*Piece{
	BlockType1: {
		blockType: BlockType1,
		blocks: toBlocks([][]int{
			{0, 0, 0, 0},
			{1, 1, 1, 1},
			{0, 0, 0, 0},
			{0, 0, 0, 0},
		}),
	},
	BlockType2: {
		blockType: BlockType2,
		blocks: toBlocks([][]int{
			{1, 0, 0},
			{1, 1, 1},
			{0, 0, 0},
		}),
	},
	BlockType3: {
		blockType: BlockType3,
		blocks: toBlocks([][]int{
			{0, 1, 0},
			{1, 1, 1},
			{0, 0, 0},
		}),
	},
	BlockType4: {
		blockType: BlockType4,
		blocks: toBlocks([][]int{
			{0, 0, 1},
			{1, 1, 1},
			{0, 0, 0},
		}),
	},
	BlockType5: {
		blockType: BlockType5,
		blocks: toBlocks([][]int{
			{1, 1, 0},
			{0, 1, 1},
			{0, 0, 0},
		}),
	},
	BlockType6: {
		blockType: BlockType6,
		blocks: toBlocks([][]int{
			{0, 1, 1},
			{1, 1, 0},
			{0, 0, 0},
		}),
	},
	BlockType7: {
		blockType: BlockType7,
		blocks: toBlocks([][]int{
			{1, 1},
			{1, 1},
		}),
	},
}

const blockWidth = 10
const blockHeight = 10
const fieldBlockNumX = 10
const fieldBlockNumY = 20

type blocksImageParts [][]BlockType

func (b blocksImageParts) Len() int {
	return len(b) * len(b[0])
}

func (b blocksImageParts) Dst(i int) (x0, y0, x1, y1 int) {
	i, j := i%len(b), i/len(b)
	x := i * blockWidth
	y := j * blockHeight
	return x, y, x + blockWidth, y + blockHeight
}

func (b blocksImageParts) Src(i int) (x0, y0, x1, y1 int) {
	i, j := i%len(b), i/len(b)
	block := b[i][j]
	if block == BlockTypeNone {
		return 0, 0, 0, 0
	}
	x := (int(block) - 1) * blockWidth
	return x, 0, x + blockWidth, blockHeight
}

func drawBlocks(r *ebiten.Image, blocks [][]BlockType, x, y int, clr ebiten.ColorM) error {
	op := &ebiten.DrawImageOptions{
		ImageParts: blocksImageParts(blocks),
		ColorM:     clr,
	}
	op.GeoM.Translate(float64(x), float64(y))
	return r.DrawImage(imageBlocks, op)
}

func (p *Piece) InitialPosition() (int, int) {
	size := len(p.blocks)
	x := (fieldBlockNumX - size) / 2
	y := 0
Loop:
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			if p.blocks[i][j] {
				break Loop
			}
		}
		y--
	}
	return x, y
}

func (p *Piece) isBlocked(i, j int, angle Angle) bool {
	size := len(p.blocks)
	i2, j2 := i, j
	switch angle {
	case Angle0:
	case Angle90:
		i2 = j
		j2 = size - 1 - i
	case Angle180:
		i2 = size - 1 - i
		j2 = size - 1 - j
	case Angle270:
		i2 = size - 1 - j
		j2 = i
	}
	return p.blocks[i2][j2]
}

func (p *Piece) collides(field *Field, x, y int, angle Angle) bool {
	size := len(p.blocks)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if field.IsBlocked(x+i, y+j) && p.isBlocked(i, j, angle) {
				return true
			}
		}
	}
	return false
}

func (p *Piece) Collides(field *Field, x, y int, angle Angle) bool {
	return p.collides(field, x, y, angle)
}

func (p *Piece) AbsorbInto(field *Field, x, y int, angle Angle) {
	size := len(p.blocks)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if p.isBlocked(i, j, angle) {
				field.setBlock(x+i, y+j, p.blockType)
			}
		}
	}
}

func (p *Piece) DrawAtCenter(r *ebiten.Image, x, y, width, height int, angle Angle) error {
	x += (width - len(p.blocks[0])*blockWidth) / 2
	y += (height - len(p.blocks)*blockHeight) / 2
	return p.Draw(r, x, y, angle)
}

func (p *Piece) Draw(r *ebiten.Image, x, y int, angle Angle) error {
	size := len(p.blocks)
	blocks := make([][]BlockType, size)
	for i := range p.blocks {
		blocks[i] = make([]BlockType, size)
		for j := range blocks[i] {
			if p.isBlocked(i, j, angle) {
				blocks[i][j] = p.blockType
			}
		}
	}
	return drawBlocks(r, blocks, x, y, ebiten.ColorM{})
}
