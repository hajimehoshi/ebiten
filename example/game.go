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

type Game struct {
	textures            map[string]graphics.TextureId
	inputStateUpdatedCh chan ui.InputStateUpdatedEvent
	x                   int
	y                   int
}

func NewGame() *Game {
	return &Game{
		textures:            map[string]graphics.TextureId{},
		inputStateUpdatedCh: make(chan ui.InputStateUpdatedEvent),
		x:                   -1,
		y:                   -1,
	}
}

func (game *Game) OnTextureCreated(e graphics.TextureCreatedEvent) {
	if e.Error != nil {
		panic(e.Error)
	}
	game.textures[e.Tag.(string)] = e.Id
}

func (game *Game) OnInputStateUpdated(e ui.InputStateUpdatedEvent) {
	go func() {
		e := e
		game.inputStateUpdatedCh <- e
	}()
}

func (game *Game) Update() {
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

func (game *Game) Draw(g graphics.Canvas) {
	if len(game.textures) < len(TexturePaths) {
		return
	}

	g.Fill(128, 128, 255)
	str := fmt.Sprintf(`Input State:
  X: %d
  Y: %d`, game.x, game.y)
	game.drawText(g, str, 5, 5)
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
	g.DrawTextureParts(game.textures["text"], parts,
		geometryMatrix, colorMatrix)
}
