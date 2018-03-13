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
)

const maxFlushCount = 20

// Field represents a game field with block states.
type Field struct {
	blocks              [fieldBlockNumX][fieldBlockNumY]BlockType
	flushCount          int
	onEndFlushAnimating func(int)
}

// IsBlocked returns a boolean value indicating whether
// there is a block at position (x, y) on the field.
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

// MovePieceToLeft tries to move the piece to the left
// and returns the piece's next x position.
func (f *Field) MovePieceToLeft(piece *Piece, x, y int, angle Angle) int {
	if piece.collides(f, x-1, y, angle) {
		return x
	}
	return x - 1
}

// MovePieceToRight tries to move the piece to the right
// and returns the piece's next x position.
func (f *Field) MovePieceToRight(piece *Piece, x, y int, angle Angle) int {
	if piece.collides(f, x+1, y, angle) {
		return x
	}
	return x + 1
}

// PieceDroppable returns a boolean value indicating whether
// the piece at (x, y) with the given angle can drop.
func (f *Field) PieceDroppable(piece *Piece, x, y int, angle Angle) bool {
	return !piece.collides(f, x, y+1, angle)
}

// DropPiece tries to drop the piece to the right
// and returns the piece's next y position.
func (f *Field) DropPiece(piece *Piece, x, y int, angle Angle) int {
	if piece.collides(f, x, y+1, angle) {
		return y
	}
	return y + 1
}

// RotatePieceRight tries to rotate the piece to the right
// and returns the piece's next angle.
func (f *Field) RotatePieceRight(piece *Piece, x, y int, angle Angle) Angle {
	if piece.collides(f, x, y, angle.RotateRight()) {
		return angle
	}
	return angle.RotateRight()
}

// RotatePieceLeft tries to rotate the piece to the left
// and returns the piece's next angle.
func (f *Field) RotatePieceLeft(piece *Piece, x, y int, angle Angle) Angle {
	if piece.collides(f, x, y, angle.RotateLeft()) {
		return angle
	}
	return angle.RotateLeft()
}

// AbsorbPiece absorbs the piece at (x, y) with the given angle into the field.
func (f *Field) AbsorbPiece(piece *Piece, x, y int, angle Angle) {
	piece.AbsorbInto(f, x, y, angle)
	if f.flushable() {
		f.flushCount = maxFlushCount
	}
}

// IsFlushAnimating returns a boolean value indicating
// whether there is a flush animation.
func (f *Field) IsFlushAnimating() bool {
	return 0 < f.flushCount
}

// SetEndFlushAnimating sets a callback fired on the end of flush animation.
// The callback argument is the number of flushed lines.
func (f *Field) SetEndFlushAnimating(fn func(lines int)) {
	f.onEndFlushAnimating = fn
}

// flushable returns a boolean value indicating whether
// there is a flushable line in the field.
func (f *Field) flushable() bool {
	for j := fieldBlockNumY - 1; 0 <= j; j-- {
		if f.flushableLine(j) {
			return true
		}
	}
	return false
}

// flushableLine returns a boolean value indicating whether
// the line j is flushabled or not.
func (f *Field) flushableLine(j int) bool {
	for i := 0; i < fieldBlockNumX; i++ {
		if f.blocks[i][j] == BlockTypeNone {
			return false
		}
	}
	return true
}

func (f *Field) setBlock(x, y int, blockType BlockType) {
	f.blocks[x][y] = blockType
}

func (f *Field) endFlushAnimating() int {
	flushedLineNum := 0
	for j := fieldBlockNumY - 1; 0 <= j; j-- {
		if f.flushLine(j + flushedLineNum) {
			flushedLineNum++
		}
	}
	return flushedLineNum
}

// flushLine flushes the line j if possible, and if the line is flushed,
// the other lines above the line go down.
//
// flushLine returns a boolean value indicating whether
// the line is actually flushed.
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

func (f *Field) Update() {
	if f.flushCount == 0 {
		return
	}

	f.flushCount--
	if f.flushCount > 0 {
		return
	}

	if f.onEndFlushAnimating != nil {
		f.onEndFlushAnimating(f.endFlushAnimating())
	}
}

func min(a, b float64) float64 {
	if a > b {
		return b
	}
	return a
}

func flushingColor(rate float64) ebiten.ColorM {
	clr := ebiten.ColorM{}
	alpha := min(1, rate*2)
	clr.Scale(1, 1, 1, alpha)
	r := min(1, (1-rate)*2)
	clr.Translate(r, 0, 0, 0)
	return clr
}

func (f *Field) Draw(r *ebiten.Image, x, y int) {
	fc := flushingColor(float64(f.flushCount) / maxFlushCount)
	for j := 0; j < fieldBlockNumY; j++ {
		if f.flushableLine(j) {
			for i := 0; i < fieldBlockNumX; i++ {
				drawBlock(r, f.blocks[i][j], i*blockWidth+x, j*blockHeight+y, fc)
			}
		} else {
			for i := 0; i < fieldBlockNumX; i++ {
				drawBlock(r, f.blocks[i][j], i*blockWidth+x, j*blockHeight+y, ebiten.ColorM{})
			}
		}
	}
}
