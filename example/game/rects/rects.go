package rects

import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image/color"
	"math"
	"math/rand"
	"time"
)

type Rects struct {
	rectTextureId     graphics.RenderTargetId
	rectTextureInited bool
	offscreenId       graphics.RenderTargetId
	offscreenInited   bool
	rectBounds        *graphics.Rect
	rectColor         *color.RGBA
}

const (
	rectTextureWidth  = 16
	rectTextureHeight = 16
	offscreenWidth    = 256
	offscreenHeight   = 240
)

func New() *Rects {
	return &Rects{
		rectTextureInited: false,
		offscreenInited:   false,
		rectBounds:        &graphics.Rect{},
		rectColor:         &color.RGBA{},
	}
}

func (game *Rects) InitTextures(tf graphics.TextureFactory) {
	var err error
	game.rectTextureId, err = tf.CreateRenderTarget(rectTextureWidth, rectTextureHeight)
	if err != nil {
		panic(err)
	}
	game.offscreenId, err = tf.CreateRenderTarget(offscreenWidth, offscreenHeight)
	if err != nil {
		panic(err)
	}
}

func (game *Rects) Update(context ebiten.GameContext) {
	game.rectBounds.X = rand.Intn(context.ScreenWidth())
	game.rectBounds.Y = rand.Intn(context.ScreenHeight())
	game.rectBounds.Width =
		rand.Intn(context.ScreenWidth()-game.rectBounds.X) + 1
	game.rectBounds.Height =
		rand.Intn(context.ScreenHeight()-game.rectBounds.Y) + 1

	game.rectColor.R = uint8(rand.Intn(math.MaxUint8))
	game.rectColor.G = uint8(rand.Intn(math.MaxUint8))
	game.rectColor.B = uint8(rand.Intn(math.MaxUint8))
	game.rectColor.A = uint8(rand.Intn(math.MaxUint8))
}

func (game *Rects) rectGeometryMatrix() matrix.Geometry {
	geometryMatrix := matrix.IdentityGeometry()
	scaleX := float64(game.rectBounds.Width) /
		float64(rectTextureWidth)
	scaleY := float64(game.rectBounds.Height) /
		float64(rectTextureHeight)
	geometryMatrix.Scale(scaleX, scaleY)
	geometryMatrix.Translate(
		float64(game.rectBounds.X), float64(game.rectBounds.Y))
	return geometryMatrix
}

func (game *Rects) rectColorMatrix() matrix.Color {
	colorMatrix := matrix.IdentityColor()
	colorMatrix.Elements[0][0] =
		float64(game.rectColor.R) / float64(math.MaxUint8)
	colorMatrix.Elements[1][1] =
		float64(game.rectColor.G) / float64(math.MaxUint8)
	colorMatrix.Elements[2][2] =
		float64(game.rectColor.B) / float64(math.MaxUint8)
	colorMatrix.Elements[3][3] =
		float64(game.rectColor.A) / float64(math.MaxUint8)
	return colorMatrix
}

func (game *Rects) Draw(g graphics.Context) {
	if !game.rectTextureInited {
		g.SetOffscreen(game.rectTextureId)
		g.Fill(255, 255, 255)
		game.rectTextureInited = true
	}

	g.SetOffscreen(game.offscreenId)
	if !game.offscreenInited {
		g.Fill(0, 0, 0)
		game.offscreenInited = true
	}
	g.DrawTexture(g.ToTexture(game.rectTextureId),
		game.rectGeometryMatrix(),
		game.rectColorMatrix())

	g.ResetOffscreen()
	g.DrawTexture(g.ToTexture(game.offscreenId),
		matrix.IdentityGeometry(),
		matrix.IdentityColor())
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
