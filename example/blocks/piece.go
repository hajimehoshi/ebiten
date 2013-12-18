package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

func init() {
	texturePaths["blocks"] = "images/blocks/blocks.png"
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
	BlockTypeMax
)

type Piece struct {
	blockType BlockType
	blocks    [][]bool
}

func toBlocks(ints [][]int) [][]bool {
	blocks := make([][]bool, len(ints))
	for i, line := range ints {
		blocks[i] = make([]bool, len(line))
		for j, v := range line {
			blocks[i][j] = v != 0
		}
	}
	return blocks
}

var Pieces = map[BlockType]*Piece{
	BlockTypeNone: nil,
	BlockType1: &Piece{
		blockType: BlockType1,
		blocks: toBlocks([][]int{
			{0, 1, 0, 0},
			{0, 1, 0, 0},
			{0, 1, 0, 0},
			{0, 1, 0, 0},
		}),
	},
	BlockType2: &Piece{
		blockType: BlockType2,
		blocks: toBlocks([][]int{
			{0, 1, 1},
			{0, 1, 0},
			{0, 1, 0},
		}),
	},
	BlockType3: &Piece{
		blockType: BlockType3,
		blocks: toBlocks([][]int{
			{0, 1, 0},
			{0, 1, 1},
			{0, 1, 0},
		}),
	},
	BlockType4: &Piece{
		blockType: BlockType4,
		blocks: toBlocks([][]int{
			{0, 1, 0},
			{0, 1, 0},
			{0, 1, 1},
		}),
	},
	BlockType5: &Piece{
		blockType: BlockType5,
		blocks: toBlocks([][]int{
			{0, 0, 1},
			{0, 1, 1},
			{0, 1, 0},
		}),
	},
	BlockType6: &Piece{
		blockType: BlockType6,
		blocks: toBlocks([][]int{
			{0, 1, 0},
			{0, 1, 1},
			{0, 0, 1},
		}),
	},
	BlockType7: &Piece{
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

func drawBlocks(context graphics.Context, blocks [][]BlockType, geo matrix.Geometry) {
	parts := []graphics.TexturePart{}
	for i, blockLine := range blocks {
		for j, block := range blockLine {
			if block == BlockTypeNone {
				continue
			}
			locationX := j * blockWidth
			locationY := i * blockHeight
			source := graphics.Rect{
				(int(block) - 1) * blockWidth, 0,
				blockWidth, blockHeight}
			parts = append(parts,
				graphics.TexturePart{
					LocationX: locationX,
					LocationY: locationY,
					Source:    source,
				})
		}
	}
	blocksTexture := drawInfo.textures["blocks"]
	context.DrawTextureParts(blocksTexture, parts, geo, matrix.IdentityColor())
}

func (p *Piece) Draw(context graphics.Context, geo matrix.Geometry) {
	blocks := make([][]BlockType, len(p.blocks))
	for i, blockLine := range p.blocks {
		blocks[i] = make([]BlockType, len(blockLine))
		for j, v := range blockLine {
			if v {
				blocks[i][j] = p.blockType
			}
		}
	}
	drawBlocks(context, blocks, geo)
}
