package main

import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/ui"
)

var TexturePaths = map[string]string{
	"ebiten": "images/ebiten.png",
	"text":   "images/text.png",
}

type drawInfo struct {
	textures   map[string]graphics.TextureId
	inputStr   string
	textureGeo matrix.Geometry
}

type Game struct {
	inputX     int
	inputY     int
	inputPrevX int
	inputPrevY int
	counter    int
	drawInfo
}

func NewGame() *Game {
	return &Game{
		inputX:     -1,
		inputY:     -1,
		inputPrevX: -1,
		inputPrevY: -1,
		counter:    0,
		drawInfo: drawInfo{
			textures:   map[string]graphics.TextureId{},
			textureGeo: matrix.IdentityGeometry(),
		},
	}
}

func (game *Game) OnTextureCreated(e graphics.TextureCreatedEvent) {
	if e.Error != nil {
		panic(e.Error)
	}
	game.textures[e.Tag.(string)] = e.Id
}

func (game *Game) OnInputStateUpdated(e ui.InputStateUpdatedEvent) {
	game.inputX, game.inputY = e.X, e.Y
}

func (game *Game) Update() {
	game.counter++
	game.drawInfo.inputStr = fmt.Sprintf(`Input State:
  X: %d
  Y: %d`, game.inputX, game.inputY)

	if game.inputPrevX != -1 && game.inputPrevY != -1 &&
		game.inputX != -1 && game.inputY != -1 {
		dx, dy := game.inputX - game.inputPrevX, game.inputY - game.inputPrevY
		game.drawInfo.textureGeo.Translate(float64(dx), float64(dy))
	}

	// Update for the next frame.
	game.inputPrevX, game.inputPrevY = game.inputX, game.inputY
}

func (game *Game) Draw(g graphics.Canvas) {
	if len(game.drawInfo.textures) < len(TexturePaths) {
		return
	}

	g.Fill(128, 128, 255)
	game.drawTexture(g, game.drawInfo.textureGeo, matrix.IdentityColor())
	game.drawText(g, game.drawInfo.inputStr, 5, 5)
}

func (game *Game) drawText(g graphics.Canvas, text string, x, y int) {
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
	g.DrawTextureParts(game.drawInfo.textures["text"], parts,
		geometryMatrix, colorMatrix)
}

func (game *Game) drawTexture(g graphics.Canvas, geo matrix.Geometry, color matrix.Color) {
	g.DrawTexture(game.drawInfo.textures["ebiten"], geo, color)
}
