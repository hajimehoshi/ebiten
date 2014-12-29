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
	"image/color"
	"math/rand"
	"time"
)

var imageEmpty *ebiten.Image

func init() {
	var err error
	imageEmpty, err = ebiten.NewImage(16, 16, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	imageEmpty.Fill(color.White)
}

func drawRect(r *ebiten.Image, x, y, width, height int) error {
	w, h := imageEmpty.Size()
	geo := ebiten.ScaleGeo(float64(width)/float64(w), float64(height)/float64(h))
	geo.Concat(ebiten.TranslateGeo(float64(x), float64(y)))
	return r.DrawImage(imageEmpty, &ebiten.DrawImageOptions{
		GeoM:   geo,
		ColorM: ebiten.ScaleColor(0.0, 0.0, 0.0, 0.5),
	})
}

type GameScene struct {
	field              *Field
	rand               *rand.Rand
	currentPiece       *Piece
	currentPieceX      int
	currentPieceY      int
	currentPieceYCarry int
	currentPieceAngle  Angle
	nextPiece          *Piece
	landingCount       int
	currentFrame       int
	lines              int
}

func NewGameScene() *GameScene {
	return &GameScene{
		field: NewField(),
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

const fieldWidth = blockWidth * fieldBlockNumX
const fieldHeight = blockHeight * fieldBlockNumY

func (s *GameScene) choosePiece() *Piece {
	num := int(BlockTypeMax)
	blockType := BlockType(s.rand.Intn(num) + 1)
	return Pieces[blockType]
}

func (s *GameScene) initCurrentPiece(piece *Piece) {
	s.currentPiece = piece
	x, y := s.currentPiece.InitialPosition()
	s.currentPieceX = x
	s.currentPieceY = y
	s.currentPieceYCarry = 0
	s.currentPieceAngle = Angle0
}

func (s *GameScene) level() int {
	return s.lines / 10
}

func (s *GameScene) Update(state *GameState) error {
	s.currentFrame++

	const maxLandingCount = 60
	if s.currentPiece == nil {
		s.initCurrentPiece(s.choosePiece())
	}
	if s.nextPiece == nil {
		s.nextPiece = s.choosePiece()
	}
	piece := s.currentPiece
	x := s.currentPieceX
	y := s.currentPieceY
	angle := s.currentPieceAngle
	moved := false
	if state.Input.StateForKey(ebiten.KeySpace) == 1 {
		s.currentPieceAngle = s.field.RotatePieceRight(piece, x, y, angle)
		moved = angle != s.currentPieceAngle
	}
	if l := state.Input.StateForKey(ebiten.KeyLeft); l == 1 || (10 <= l && l%2 == 0) {
		s.currentPieceX = s.field.MovePieceToLeft(piece, x, y, angle)
		moved = x != s.currentPieceX
	}
	if r := state.Input.StateForKey(ebiten.KeyRight); r == 1 || (10 <= r && r%2 == 0) {
		s.currentPieceX = s.field.MovePieceToRight(piece, x, y, angle)
		moved = y != s.currentPieceX
	}
	if d := state.Input.StateForKey(ebiten.KeyDown); (d-1)%2 == 0 {
		s.currentPieceY = s.field.DropPiece(piece, x, y, angle)
		moved = y != s.currentPieceY
	}

	// Drop the current piece with gravity.
	s.currentPieceYCarry += 2*s.level() + 1
	for 60 <= s.currentPieceYCarry {
		s.currentPieceYCarry -= 60
		s.currentPieceY = s.field.DropPiece(piece, s.currentPieceX, s.currentPieceY, angle)
		moved = y != s.currentPieceY
	}

	if moved {
		s.landingCount = 0
	} else if !s.field.PieceDroppable(piece, s.currentPieceX, s.currentPieceY, angle) {
		if 0 < state.Input.StateForKey(ebiten.KeyDown) {
			s.landingCount += 10
		} else {
			s.landingCount++
		}
		if maxLandingCount <= s.landingCount {
			lines := s.field.AbsorbPiece(piece, s.currentPieceX, s.currentPieceY, angle)
			s.lines += lines
			s.initCurrentPiece(s.nextPiece)
			s.nextPiece = nil
			s.landingCount = 0
		}
	}
	return nil
}

func (s *GameScene) Draw(r *ebiten.Image) error {
	if err := r.Fill(color.White); err != nil {
		return err
	}

	const fieldX, fieldY = 20, 20

	// Draw field
	if err := drawRect(r, fieldX, fieldY, fieldWidth, fieldHeight); err != nil {
		return err
	}

	// Draw next
	x := fieldX + fieldWidth + blockWidth*2
	y := fieldY
	if err := drawTextWithShadow(r, "NEXT", x, y, 1, color.NRGBA{0x00, 0x00, 0x80, 0xff}); err != nil {
		return err
	}
	nextX := x
	nextY := y + blockHeight
	if err := drawRect(r, nextX, nextY, blockWidth*5, blockHeight*5); err != nil {
		return err
	}

	if err := s.field.Draw(r, fieldX, fieldY); err != nil {
		return err
	}
	if s.currentPiece != nil {
		x := fieldX + s.currentPieceX*blockWidth
		y := fieldY + s.currentPieceY*blockHeight
		if err := s.currentPiece.Draw(r, x, y, s.currentPieceAngle); err != nil {
			return err
		}
	}
	if s.nextPiece != nil {
		x := nextX
		y := nextY
		if err := s.nextPiece.DrawAtCenter(r, x, y, blockWidth*5, blockHeight*5, 0); err != nil {
			return err
		}
	}
	return nil
}
