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
	rectTextureId       graphics.RenderTargetId
	rectTextureInited   bool
	offscreenId         graphics.RenderTargetId
	offscreenInited     bool
	rectBounds          *graphics.Rect
	rectColor           *color.RGBA
	screenSizeUpdatedCh chan ebiten.ScreenSizeUpdatedEvent
	screenWidth         int
	screenHeight        int
}

const (
	rectTextureWidth  = 16
	rectTextureHeight = 16
	offscreenWidth    = 256
	offscreenHeight   = 240
)

func New() *Rects {
	return &Rects{
		rectTextureInited:   false,
		offscreenInited:     false,
		rectBounds:          &graphics.Rect{},
		rectColor:           &color.RGBA{},
		screenSizeUpdatedCh: make(chan ebiten.ScreenSizeUpdatedEvent),
	}
}

func (game *Rects) OnScreenSizeUpdated(e ebiten.ScreenSizeUpdatedEvent) {
	go func() {
		e := e
		game.screenSizeUpdatedCh <- e
	}()
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (game *Rects) Update() {
events:
	for {
		select {
		case e := <-game.screenSizeUpdatedCh:
			game.screenWidth, game.screenHeight = e.Width, e.Height
		default:
			break events
		}
	}

	if game.screenWidth == 0 || game.screenHeight == 0 {
		return
	}

	x1 := rand.Intn(game.screenWidth)
	x2 := rand.Intn(game.screenWidth)
	y1 := rand.Intn(game.screenHeight)
	y2 := rand.Intn(game.screenHeight)
	game.rectBounds.X = min(x1, x2)
	game.rectBounds.Y = min(y1, y2)
	game.rectBounds.Width = abs(x1 - x2)
	game.rectBounds.Height = abs(y1 - y2)

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

func (game *Rects) Draw(g graphics.Canvas) {
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
	g.DrawRenderTarget(game.rectTextureId,
		game.rectGeometryMatrix(),
		game.rectColorMatrix())

	g.ResetOffscreen()
	g.DrawRenderTarget(game.offscreenId,
		matrix.IdentityGeometry(),
		matrix.IdentityColor())
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
