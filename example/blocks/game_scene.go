package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image/color"
	"math/rand"
	"time"
)

func init() {
	texturePaths["empty"] = "images/blocks/empty.png"
}

type GameScene struct {
	field        *Field
	rand         *rand.Rand
	currentPiece *Piece
	nextPiece    *Piece
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
	// Omit BlockTypeNone.
	num := int(BlockTypeMax) - 1
	blockType := BlockType(s.rand.Intn(num) + 1)
	return Pieces[blockType]
}

func (s *GameScene) Update(state *GameState) {
	if s.currentPiece == nil {
		s.currentPiece = s.choosePiece()
	}
}

func (s *GameScene) Draw(context graphics.Context) {
	context.Fill(0xff, 0xff, 0xff)

	field := drawInfo.textures["empty"]
	geoMat := matrix.IdentityGeometry()
	geoMat.Scale(float64(fieldWidth) / float64(emptyWidth),
		float64(fieldHeight) / float64(emptyHeight))
	geoMat.Translate(20, 20) // magic number?
	colorMat := matrix.IdentityColor()
	colorMat.Scale(color.RGBA{0, 0, 0, 0x80})
	context.DrawTexture(field, geoMat, colorMat)

	geoMat = matrix.IdentityGeometry()
	geoMat.Translate(20, 20)
	s.field.Draw(context, geoMat)

	if s.currentPiece != nil {
		s.currentPiece.Draw(context, geoMat)
	}
}
