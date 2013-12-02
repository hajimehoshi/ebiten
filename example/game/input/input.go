package input

import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/ui"
	"image"
	"os"
)

type Input struct {
	textTextureId       graphics.TextureId
	inputStateUpdatedCh chan ui.InputStateUpdatedEvent
	x                   int
	y                   int
}

func New() *Input {
	return &Input{
		inputStateUpdatedCh: make(chan ui.InputStateUpdatedEvent),
		x:                   -1,
		y:                   -1,
	}
}

func (game *Input) OnInputStateUpdated(e ui.InputStateUpdatedEvent) {
	go func() {
		e := e
		game.inputStateUpdatedCh <- e
	}()
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
	if game.textTextureId, err = tf.CreateTextureFromImage(img); err != nil {
		panic(err)
	}
}

func (game *Input) Update() {
events:
	for {
		select {
		case e := <-game.inputStateUpdatedCh:
			game.x, game.y = e.X, e.Y
		default:
			break events
		}
	}
}

func (game *Input) Draw(g graphics.Canvas) {
	g.Fill(128, 128, 255)
	str := fmt.Sprintf(`Input State:
  X: %d
  Y: %d`, game.x, game.y)
	game.drawText(g, str, 5, 5)
}

func (game *Input) drawText(g graphics.Canvas, text string, x, y int) {
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
	g.DrawTextureParts(game.textTextureId, parts,
		geometryMatrix, colorMatrix)
}
