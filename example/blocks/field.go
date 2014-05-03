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

func (f *Field) IsBlocked(x, y int) bool {
	if x < 0 || fieldBlockNumX <= x {
		return true
	}
	if y < 0 {
		return false
	}
	if fieldBlockNumY <= y {
		return true
	}
	return f.blocks[x][y] != BlockTypeNone
}

func (f *Field) collides(piece *Piece, x, y int, angle Angle) bool {
	return piece.collides(f, x, y, angle)
}

func (f *Field) MovePieceToLeft(piece *Piece, x, y int, angle Angle) int {
	if f.collides(piece, x-1, y, angle) {
		return x
	}
	return x - 1
}

func (f *Field) MovePieceToRight(piece *Piece, x, y int, angle Angle) int {
	if f.collides(piece, x+1, y, angle) {
		return x
	}
	return x + 1
}

func (f *Field) PieceDroppable(piece *Piece, x, y int, angle Angle) bool {
	return !f.collides(piece, x, y+1, angle)
}

func (f *Field) DropPiece(piece *Piece, x, y int, angle Angle) int {
	if f.collides(piece, x, y+1, angle) {
		return y
	}
	return y + 1
}

func (f *Field) RotatePieceRight(piece *Piece, x, y int, angle Angle) Angle {
	if f.collides(piece, x, y, angle.RotateRight()) {
		return angle
	}
	return angle.RotateRight()
}

func (f *Field) AbsorbPiece(piece *Piece, x, y int, angle Angle) {
	piece.AbsorbInto(f, x, y, angle)
	f.Flush()
}

func (f *Field) flushLine(j int) bool {
	for i := 0; i < fieldBlockNumX; i++ {
		if f.blocks[i][j] == BlockTypeNone {
			return false
		}
	}
	for j2 := j; 1 <= j2; j2-- {
		for i := 0; i < fieldBlockNumX; i++ {
			f.blocks[i][j2] = f.blocks[i][j2-1]
		}
	}
	for i := 0; i < fieldBlockNumX; i++ {
		f.blocks[i][0] = BlockTypeNone
	}
	return true
}

func (f *Field) Flush() {
	flushedLineNum := 0
	for j := fieldBlockNumY - 1; 0 <= j; j-- {
		if f.flushLine(j + flushedLineNum) {
			flushedLineNum++
		}
	}
}

func (f *Field) Draw(context graphics.Context, textures *Textures, geo matrix.Geometry) {
	blocks := make([][]BlockType, len(f.blocks))
	for i, blockCol := range f.blocks {
		blocks[i] = make([]BlockType, len(blockCol))
		copy(blocks[i], blockCol[:])
	}
	drawBlocks(context, textures, blocks, geo)
}
