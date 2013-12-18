package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Field struct {
	blocks [fieldBlockNumX][fieldBlockNumY]BlockType
}

func NewField() *Field {
	return &Field{}
}

func (f *Field) Draw(context graphics.Context, geo matrix.Geometry) {
	blocks := make([][]BlockType, len(f.blocks))
	for i, blockLine := range f.blocks {
		blocks[i] = make([]BlockType, len(blockLine))
		copy(blocks[i], blockLine[:])
	}
	drawBlocks(context, blocks, geo)
}
