package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image/color"
	"math/rand"
	"time"
)

func init() {
	texturePaths["empty"] = "images/blocks/empty.png"
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

const emptyWidth = 16
const emptyHeight = 16
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
	if state.Input.StateForKey(ui.KeySpace) == 1 {
		s.currentPieceAngle = s.field.RotatePieceRight(piece, x, y, angle)
		moved = angle != s.currentPieceAngle
	}
	if l := state.Input.StateForKey(ui.KeyLeft); l == 1 || (10 <= l && l%2 == 0) {
		s.currentPieceX = s.field.MovePieceToLeft(piece, x, y, angle)
		moved = x != s.currentPieceX
	}
	if r := state.Input.StateForKey(ui.KeyRight); r == 1 || (10 <= r && r%2 == 0) {
		s.currentPieceX = s.field.MovePieceToRight(piece, x, y, angle)
		moved = y != s.currentPieceX
	}
	if d := state.Input.StateForKey(ui.KeyDown); (d-1)%2 == 0 {
		s.currentPieceY = s.field.DropPiece(piece, x, y, angle)
		moved = y != s.currentPieceY
	}
	if moved {
		s.landingCount = 0
	} else if !s.field.PieceDroppable(piece, x, y, angle) {
		if 0 < state.Input.StateForKey(ui.KeyDown) {
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

func (s *GameScene) Draw(context graphics.Context, textures Textures) {
	context.Fill(0xff, 0xff, 0xff)

	field := textures.GetTexture("empty")
	geoMat := matrix.IdentityGeometry()
	geoMat.Scale(
		float64(fieldWidth)/float64(emptyWidth),
		float64(fieldHeight)/float64(emptyHeight))
	geoMat.Translate(20, 20) // magic number?
	colorMat := matrix.IdentityColor()
	colorMat.Scale(color.RGBA{0, 0, 0, 0x80})
	graphics.DrawWhole(
		context.Texture(field),
		emptyWidth,
		emptyHeight,
		geoMat,
		colorMat)

	geoMat = matrix.IdentityGeometry()
	geoMat.Translate(20, 20)
	s.field.Draw(context, textures, geoMat)

	if s.currentPiece != nil {
		s.currentPiece.Draw(
			context,
			textures,
			20, 20,
			s.currentPieceX,
			s.currentPieceY,
			s.currentPieceAngle)
	}
}
