package input

import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
	"os"
)

type Input struct {
	textTextureID graphics.TextureID
	inputState    ebiten.InputState
}

func New() *Input {
	return &Input{}
}

func (game *Input) InitTextures(tf graphics.TextureFactory) {
	file, err := os.Open("images/text.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	if game.textTextureID, err = tf.NewTextureFromImage(img); err != nil {
		panic(err)
	}
}

func (game *Input) Update(context ebiten.GameContext) {
	game.inputState = context.InputState()
}

func (game *Input) Draw(g graphics.Context) {
	g.Fill(128, 128, 255)
	str := fmt.Sprintf(`Input State:
  X: %d
  Y: %d`, game.inputState.X, game.inputState.Y)
	game.drawText(g, str, 5, 5)
}

func (game *Input) drawText(g graphics.Context, text string, x, y int) {
	const letterWidth = 6
	const letterHeight = 16

	parts := []graphics.TexturePart{}
	textX := 0
	textY := 0
	for _, c := range text {
		if c == '\n' {
			textX = 0
			textY += letterHeight
			continue
		}
		code := int(c)
		x := (code % 32) * letterWidth
		y := (code / 32) * letterHeight
		source := graphics.Rect{x, y, letterWidth, letterHeight}
		parts = append(parts, graphics.TexturePart{
			LocationX: textX,
			LocationY: textY,
			Source:    source,
		})
		textX += letterWidth
	}

	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Translate(float64(x), float64(y))
	colorMatrix := matrix.IdentityColor()
	g.DrawTextureParts(game.textTextureID, parts,
		geometryMatrix, colorMatrix)
}
