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
	var err error
	// Background
	imageGameBG, _, err = ebitenutil.NewImageFromFile("images/gophers.jpg", ebiten.FilterLinear)
	if err != nil {
		panic(err)
	}

	// Windows
	imageWindows, err = ebiten.NewImage(ScreenWidth, ScreenHeight, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	// Windows: Field
	x, y := fieldWindowPosition()
	if err := drawWindow(imageWindows, x, y, fieldWidth, fieldHeight); err != nil {
		panic(err)
	}
	// Windows: Next
	x, y = nextWindowLabelPosition()
	if err := drawTextWithShadow(imageWindows, "NEXT", x, y, 1, fontColor); err != nil {
		panic(err)
	}
	x, y = nextWindowPosition()
	if err := drawWindow(imageWindows, x, y, 5*blockWidth, 5*blockHeight); err != nil {
		panic(err)
	}
	// Windows: Score
	x, y = scoreTextBoxPosition()
	if err := drawTextBox(imageWindows, "SCORE", x, y, textBoxWidth()); err != nil {
		panic(err)
	}
	// Windows: Level
	x, y = levelTextBoxPosition()
	if err := drawTextBox(imageWindows, "LEVEL", x, y, textBoxWidth()); err != nil {
		panic(err)
	}
	// Windows: Lines
	x, y = linesTextBoxPosition()
	if err := drawTextBox(imageWindows, "LINES", x, y, textBoxWidth()); err != nil {
		panic(err)
	}

	// Gameover
	imageGameover, err = ebiten.NewImage(ScreenWidth, ScreenHeight, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	imageGameover.Fill(color.NRGBA{0x00, 0x00, 0x00, 0x80})
	y = (ScreenHeight - blockHeight) / 2
	if err := drawTextWithShadowCenter(imageGameover, "GAME OVER", 0, y, 1, color.White, ScreenWidth); err != nil {
		panic(err)
	}
}

func drawWindow(r *ebiten.Image, x, y, width, height int) error {
	return r.DrawFilledRect(x, y, width, height, color.NRGBA{0, 0, 0, 0xc0})
}

var fontColor = color.NRGBA{0x40, 0x40, 0xff, 0xff}

func drawTextBox(r *ebiten.Image, label string, x, y, width int) error {
	if err := drawTextWithShadow(r, label, x, y, 1, fontColor); err != nil {
		return err
	}
	y += blockWidth
	if err := drawWindow(r, x, y, width, 2*blockHeight); err != nil {
		return err
	}
	return nil
}

func drawTextBoxContent(r *ebiten.Image, content string, x, y, width int) error {
	y += blockWidth
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
		if state.Input.StateForKey(ebiten.KeySpace) == 1 || state.Input.StateForKey(ebiten.KeyX) == 1 {
			s.currentPieceAngle = s.field.RotatePieceRight(piece, x, y, angle)
			moved = angle != s.currentPieceAngle
		}
		if state.Input.StateForKey(ebiten.KeyZ) == 1 {
			s.currentPieceAngle = s.field.RotatePieceLeft(piece, x, y, angle)
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

	if err := r.DrawImage(imageWindows, nil); err != nil {
		return err
	}

	// Draw score
	x, y := scoreTextBoxPosition()
	if err := drawTextBoxContent(r, strconv.Itoa(s.score), x, y, textBoxWidth()); err != nil {
		return err
	}

	// Draw level
	x, y = levelTextBoxPosition()
	if err := drawTextBoxContent(r, strconv.Itoa(s.level()), x, y, textBoxWidth()); err != nil {
		return err
	}

	// Draw lines
	x, y = linesTextBoxPosition()
	if err := drawTextBoxContent(r, strconv.Itoa(s.lines), x, y, textBoxWidth()); err != nil {
		return err
	}

	// Draw blocks
	fieldX, fieldY := fieldWindowPosition()
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
		// TODO: Make functions to get these values.
		x := fieldX + fieldWidth + blockWidth*2
		y := fieldY + blockHeight
		if err := s.nextPiece.DrawAtCenter(r, x, y, blockWidth*5, blockHeight*5, 0); err != nil {
			return err
		}
	}

	if s.gameover {
		r.DrawImage(imageGameover, nil)
	}

	return nil
}
