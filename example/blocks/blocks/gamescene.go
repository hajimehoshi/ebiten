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
	"image/color"
	_ "image/jpeg"
	"math/rand"
	"strconv"
	"time"
)

var (
	imageEmpty    *ebiten.Image
	imageGameBG   *ebiten.Image
	imageGameover *ebiten.Image
)

func init() {
	var err error
	imageEmpty, err = ebiten.NewImage(16, 16, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	imageEmpty.Fill(color.White)

	imageGameBG, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterLinear)
	if err != nil {
		panic(err)
	}

	imageGameover, err = ebiten.NewImage(ScreenWidth, ScreenHeight, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	imageGameover.Fill(color.NRGBA{0x00, 0x00, 0x00, 0x80})
	y := (ScreenHeight - blockHeight) / 2
	if err := drawTextWithShadowCenter(imageGameover, "GAME OVER", 0, y, 1, color.White, ScreenWidth); err != nil {
		panic(err)
	}
}

func drawRect(r *ebiten.Image, x, y, width, height int) error {
	w, h := imageEmpty.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(width)/float64(w), float64(height)/float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorM.Scale(0.0, 0.0, 0.0, 0.75)
	return r.DrawImage(imageEmpty, op)
}

var fontColor = color.NRGBA{0x40, 0x40, 0xff, 0xff}

func drawTextBox(r *ebiten.Image, label, content string, x, y, width int) error {
	if err := drawTextWithShadow(r, label, x, y, 1, fontColor); err != nil {
		return err
	}
	y += blockWidth
	if err := drawRect(r, x, y, width, 2*blockHeight); err != nil {
		return err
	}
	if err := drawTextWithShadowRight(r, content, x, y+blockHeight*3/4, 1, color.White, width-blockWidth/2); err != nil {
		return err
	}
	return nil
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
	score              int
	lines              int
	gameover           bool
}

func NewGameScene() *GameScene {
	return &GameScene{
		field: NewField(),
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *GameScene) drawBackground(r *ebiten.Image) error {
	if err := r.Fill(color.White); err != nil {
		return err
	}

	w, h := imageGameBG.Size()
	scaleW := ScreenWidth / float64(w)
	scaleH := ScreenHeight / float64(h)
	scale := scaleW
	if scale < scaleH {
		scale = scaleH
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(ScreenWidth/2, ScreenHeight/2)

	a := 0.7
	m := ebiten.Monochrome()
	m.Scale(a, a, a, a)
	op.ColorM.Scale(1-a, 1-a, 1-a, 1-a)
	op.ColorM.Add(m)
	op.ColorM.Translate(0.3, 0.3, 0.3, 0)
	return r.DrawImage(imageGameBG, op)
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

func (s *GameScene) addScore(lines int) {
	base := 0
	switch lines {
	case 1:
		base = 100
	case 2:
		base = 300
	case 3:
		base = 600
	case 4:
		base = 1000
	default:
		panic("not reach")
	}
	s.score += (s.level() + 1) * base
}

func (s *GameScene) Update(state *GameState) error {
	s.field.Update()

	if s.gameover {
		if state.Input.StateForKey(ebiten.KeySpace) == 1 {
			state.SceneManager.GoTo(NewTitleScene())
		}
		return nil
	}

	s.currentFrame++

	const maxLandingCount = 60
	if s.currentPiece == nil {
		s.initCurrentPiece(s.choosePiece())
	}
	if s.nextPiece == nil {
		s.nextPiece = s.choosePiece()
	}

	moved := false
	piece := s.currentPiece
	angle := s.currentPieceAngle

	// Move piece by user input.
	if !s.field.Flushing() {
		piece := s.currentPiece
		x := s.currentPieceX
		y := s.currentPieceY
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
			if moved {
				s.score++
			}
		}
	}

	// Drop the current piece with gravity.
	if !s.field.Flushing() {
		y := s.currentPieceY
		angle := s.currentPieceAngle
		s.currentPieceYCarry += 2*s.level() + 1
		for 60 <= s.currentPieceYCarry {
			s.currentPieceYCarry -= 60
			s.currentPieceY = s.field.DropPiece(piece, s.currentPieceX, s.currentPieceY, angle)
			moved = y != s.currentPieceY
		}
	}

	if moved {
		s.landingCount = 0
	} else if !s.field.Flushing() && !s.field.PieceDroppable(piece, s.currentPieceX, s.currentPieceY, angle) {
		if 0 < state.Input.StateForKey(ebiten.KeyDown) {
			s.landingCount += 10
		} else {
			s.landingCount++
		}
		if maxLandingCount <= s.landingCount {
			s.field.AbsorbPiece(piece, s.currentPieceX, s.currentPieceY, angle)
			if s.field.Flushing() {
				s.field.SetEndFlushing(func(lines int) {
					s.lines += lines
					if 0 < lines {
						s.addScore(lines)
					}
					s.goNextPiece()
				})
			} else {
				s.goNextPiece()
			}

		}
	}
	return nil
}

func (s *GameScene) goNextPiece() {
	s.initCurrentPiece(s.nextPiece)
	s.nextPiece = s.choosePiece()
	s.landingCount = 0
	if s.currentPiece.Collides(s.field, s.currentPieceX, s.currentPieceY, s.currentPieceAngle) {
		s.gameover = true
	}
}

func (s *GameScene) Draw(r *ebiten.Image) error {
	if err := s.drawBackground(r); err != nil {
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
	if err := drawTextWithShadow(r, "NEXT", x, y, 1, fontColor); err != nil {
		return err
	}
	nextX := x
	nextY := y + blockHeight
	if err := drawRect(r, nextX, nextY, 5*blockWidth, 5*blockHeight); err != nil {
		return err
	}
	x = nextX
	y = nextY + 5*blockHeight + blockHeight

	// Draw score
	width := ScreenWidth - 2*blockWidth - x
	if err := drawTextBox(r, "SCORE", strconv.Itoa(s.score), x, y, width); err != nil {
		return err
	}

	// Draw level
	y += 4 * blockHeight
	if err := drawTextBox(r, "LEVEL", strconv.Itoa(s.level()), x, y, width); err != nil {
		return err
	}

	// Draw lines
	y += 4 * blockHeight
	if err := drawTextBox(r, "LINES", strconv.Itoa(s.lines), x, y, width); err != nil {
		return err
	}

	// Draw blocks
	if err := s.field.Draw(r, fieldX, fieldY); err != nil {
		return err
	}
	if s.currentPiece != nil && !s.field.Flushing() {
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

	if s.gameover {
		r.DrawImage(imageGameover, nil)
	}

	return nil
}
