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

func init() {
	imagePaths["empty"] = "images/blocks/empty.png"
}

type GameScene struct {
	field             *Field
	rand              *rand.Rand
	currentPiece      *Piece
	currentPieceX     int
	currentPieceY     int
	currentPieceAngle Angle
	nextPiece         *Piece
	landingCount      int
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
	num := NormalBlockTypeNum
	blockType := BlockType(s.rand.Intn(num) + 1)
	return Pieces[blockType]
}

func (s *GameScene) initCurrentPiece(piece *Piece) {
	s.currentPiece = piece
	x, y := s.currentPiece.InitialPosition()
	s.currentPieceX = x
	s.currentPieceY = y
	s.currentPieceAngle = Angle0
}

func (s *GameScene) Update(state *GameState) {
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
	if moved {
		s.landingCount = 0
	} else if !s.field.PieceDroppable(piece, x, y, angle) {
		if 0 < state.Input.StateForKey(ebiten.KeyDown) {
			s.landingCount += 10
		} else {
			s.landingCount++
		}
		if maxLandingCount <= s.landingCount {
			s.field.AbsorbPiece(piece, x, y, angle)
			s.initCurrentPiece(s.nextPiece)
			s.nextPiece = nil
			s.landingCount = 0
		}
	}
}

func (s *GameScene) Draw(r *ebiten.Image, images *Images) {
	r.Fill(color.White)

	field := images.GetImage("empty")
	w, h := field.Size()
	geo := ebiten.ScaleGeometry(float64(fieldWidth)/float64(w), float64(fieldHeight)/float64(h))
	geo.Concat(ebiten.TranslateGeometry(20, 20)) // TODO: magic number?
	clr := ebiten.ScaleColor(0.0, 0.0, 0.0, 0.5)
	r.DrawImage(field, &ebiten.ImageDrawOption{
		GeometryMatrix: &geo,
		ColorMatrix:    &clr,
	})

	s.field.Draw(r, images, 20, 20)

	if s.currentPiece != nil {
		s.currentPiece.Draw(r, images, 20, 20, s.currentPieceX, s.currentPieceY, s.currentPieceAngle)
	}
}
