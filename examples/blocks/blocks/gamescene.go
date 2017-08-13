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

// +build example

package blocks

import (
	"image/color"
	_ "image/jpeg"
	"math/rand"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/common"
)

var (
	imageGameBG   *ebiten.Image
	imageWindows  *ebiten.Image
	imageGameover *ebiten.Image
)

func fieldWindowPosition() (x, y int) {
	return 20, 20
}

func nextWindowLabelPosition() (x, y int) {
	x, y = fieldWindowPosition()
	return x + fieldWidth + 2*blockWidth, y
}

func nextWindowPosition() (x, y int) {
	x, y = nextWindowLabelPosition()
	return x, y + blockHeight
}

func textBoxWidth() int {
	x, _ := nextWindowPosition()
	return ScreenWidth - 2*blockWidth - x
}

func scoreTextBoxPosition() (x, y int) {
	x, y = nextWindowPosition()
	return x, y + 6*blockHeight
}

func levelTextBoxPosition() (x, y int) {
	x, y = scoreTextBoxPosition()
	return x, y + 4*blockHeight
}

func linesTextBoxPosition() (x, y int) {
	x, y = levelTextBoxPosition()
	return x, y + 4*blockHeight
}

func init() {
	// Background
	var err error
	imageGameBG, _, err = ebitenutil.NewImageFromFile("_resources/images/gophers.jpg", ebiten.FilterLinear)
	if err != nil {
		panic(err)
	}

	// Windows
	imageWindows, _ = ebiten.NewImage(ScreenWidth, ScreenHeight, ebiten.FilterNearest)
	// Windows: Field
	x, y := fieldWindowPosition()
	drawWindow(imageWindows, x, y, fieldWidth, fieldHeight)
	// Windows: Next
	x, y = nextWindowLabelPosition()
	common.ArcadeFont.DrawTextWithShadow(imageWindows, "NEXT", x, y, 1, fontColor)
	x, y = nextWindowPosition()
	drawWindow(imageWindows, x, y, 5*blockWidth, 5*blockHeight)
	// Windows: Score
	x, y = scoreTextBoxPosition()
	drawTextBox(imageWindows, "SCORE", x, y, textBoxWidth())
	// Windows: Level
	x, y = levelTextBoxPosition()
	drawTextBox(imageWindows, "LEVEL", x, y, textBoxWidth())
	// Windows: Lines
	x, y = linesTextBoxPosition()
	drawTextBox(imageWindows, "LINES", x, y, textBoxWidth())

	// Gameover
	imageGameover, _ = ebiten.NewImage(ScreenWidth, ScreenHeight, ebiten.FilterNearest)
	imageGameover.Fill(color.NRGBA{0x00, 0x00, 0x00, 0x80})
	y = (ScreenHeight - blockHeight) / 2
	drawTextWithShadowCenter(imageGameover, "GAME OVER", 0, y, 1, color.White, ScreenWidth)
}

func drawWindow(r *ebiten.Image, x, y, width, height int) {
	ebitenutil.DrawRect(r, float64(x), float64(y), float64(width), float64(height), color.RGBA{0, 0, 0, 0xc0})
}

var fontColor = color.NRGBA{0x40, 0x40, 0xff, 0xff}

func drawTextBox(r *ebiten.Image, label string, x, y, width int) {
	common.ArcadeFont.DrawTextWithShadow(r, label, x, y, 1, fontColor)
	y += blockWidth
	drawWindow(r, x, y, width, 2*blockHeight)
}

func drawTextBoxContent(r *ebiten.Image, content string, x, y, width int) {
	y += blockWidth
	drawTextWithShadowRight(r, content, x, y+blockHeight*3/4, 1, color.White, width-blockWidth/2)
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

func (s *GameScene) drawBackground(r *ebiten.Image) {
	r.Fill(color.White)

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
	r.DrawImage(imageGameBG, op)
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
		// TODO: Gamepad key?
		if state.Input.StateForKey(ebiten.KeySpace) == 1 {
			state.SceneManager.GoTo(NewTitleScene())
		}
		return nil
	}

	s.currentFrame++

	const maxLandingCount = ebiten.FPS
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
		if state.Input.IsRotateRightTrigger() {
			s.currentPieceAngle = s.field.RotatePieceRight(piece, x, y, angle)
			moved = angle != s.currentPieceAngle
		} else if state.Input.IsRotateLeftTrigger() {
			s.currentPieceAngle = s.field.RotatePieceLeft(piece, x, y, angle)
			moved = angle != s.currentPieceAngle
		} else if l := state.Input.StateForLeft(); l == 1 || (10 <= l && l%2 == 0) {
			s.currentPieceX = s.field.MovePieceToLeft(piece, x, y, angle)
			moved = x != s.currentPieceX
		} else if r := state.Input.StateForRight(); r == 1 || (10 <= r && r%2 == 0) {
			s.currentPieceX = s.field.MovePieceToRight(piece, x, y, angle)
			moved = y != s.currentPieceX
		} else if d := state.Input.StateForDown(); (d-1)%2 == 0 {
			s.currentPieceY = s.field.DropPiece(piece, x, y, angle)
			moved = y != s.currentPieceY
			if moved {
				s.score++
			}
		}
	}

	// Drop the current piece with gravity.
	if !s.field.Flushing() {
		angle := s.currentPieceAngle
		s.currentPieceYCarry += 2*s.level() + 1
		const maxCarry = 60
		for maxCarry <= s.currentPieceYCarry {
			s.currentPieceYCarry -= maxCarry
			s.currentPieceY = s.field.DropPiece(piece, s.currentPieceX, s.currentPieceY, angle)
		}
	}

	if !s.field.Flushing() && !s.field.PieceDroppable(piece, s.currentPieceX, s.currentPieceY, angle) {
		if 0 < state.Input.StateForDown() {
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

func (s *GameScene) Draw(r *ebiten.Image) {
	s.drawBackground(r)

	r.DrawImage(imageWindows, nil)

	// Draw score
	x, y := scoreTextBoxPosition()
	drawTextBoxContent(r, strconv.Itoa(s.score), x, y, textBoxWidth())

	// Draw level
	x, y = levelTextBoxPosition()
	drawTextBoxContent(r, strconv.Itoa(s.level()), x, y, textBoxWidth())

	// Draw lines
	x, y = linesTextBoxPosition()
	drawTextBoxContent(r, strconv.Itoa(s.lines), x, y, textBoxWidth())

	// Draw blocks
	fieldX, fieldY := fieldWindowPosition()
	s.field.Draw(r, fieldX, fieldY)
	if s.currentPiece != nil && !s.field.Flushing() {
		x := fieldX + s.currentPieceX*blockWidth
		y := fieldY + s.currentPieceY*blockHeight
		s.currentPiece.Draw(r, x, y, s.currentPieceAngle)
	}
	if s.nextPiece != nil {
		// TODO: Make functions to get these values.
		x := fieldX + fieldWidth + blockWidth*2
		y := fieldY + blockHeight
		s.nextPiece.DrawAtCenter(r, x, y, blockWidth*5, blockHeight*5, 0)
	}

	if s.gameover {
		r.DrawImage(imageGameover, nil)
	}
}
