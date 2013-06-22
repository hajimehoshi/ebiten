package rects

import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image/color"
	"math/rand"
	"time"
)

type Rects struct {
	rectsTexture graphics.Texture
}

func New() *Rects {
	return &Rects{}
}

func (game *Rects) ScreenWidth() int {
	return 256
}

func (game *Rects) ScreenHeight() int {
	return 240
}

func (game *Rects) Init(tf graphics.TextureFactory) {
	game.rectsTexture = tf.NewTexture(game.ScreenWidth(), game.ScreenHeight())
}

func (game *Rects) Update() {
}

func (game *Rects) Draw(g graphics.GraphicsContext, offscreen graphics.Texture) {
	g.SetOffscreen(game.rectsTexture.ID)

	x := rand.Intn(game.ScreenWidth())
	y := rand.Intn(game.ScreenHeight())
	width := rand.Intn(game.ScreenWidth() - x)
	height := rand.Intn(game.ScreenHeight() - y)

	red := uint8(rand.Intn(256))
	green := uint8(rand.Intn(256))
	blue := uint8(rand.Intn(256))
	alpha := uint8(rand.Intn(256))

	g.DrawRect(
		graphics.Rect{x, y, width, height},
		&color.RGBA{red, green, blue, alpha},
	)

	g.SetOffscreen(offscreen.ID)
	g.DrawTexture(game.rectsTexture,
		matrix.IdentityGeometry(),
		matrix.IdentityColor())
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
