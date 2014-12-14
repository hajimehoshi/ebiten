package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image/color"
	"log"
	"math"
	"runtime"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	count              int
	brushRenderTarget  ebiten.RenderTargetID
	canvasRenderTarget ebiten.RenderTargetID
}

func (g *Game) Update() error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.count++
	}
	return nil
}

func (g *Game) Draw(gr ebiten.GraphicsContext) error {
	if g.brushRenderTarget.IsNil() {
		var err error
		g.brushRenderTarget, err = ebiten.NewRenderTargetID(1, 1, ebiten.FilterNearest)
		if err != nil {
			return err
		}

		gr.PushRenderTarget(g.brushRenderTarget)
		gr.Fill(0xff, 0xff, 0xff)
		gr.PopRenderTarget()
	}
	if g.canvasRenderTarget.IsNil() {
		var err error
		g.canvasRenderTarget, err = ebiten.NewRenderTargetID(screenWidth, screenHeight, ebiten.FilterNearest)
		if err != nil {
			return err
		}
		gr.PushRenderTarget(g.canvasRenderTarget)
		gr.Fill(0xff, 0xff, 0xff)
		gr.PopRenderTarget()
	}
	mx, my := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		gr.PushRenderTarget(g.canvasRenderTarget)
		geo := ebiten.GeometryMatrixI()
		geo.Concat(ebiten.TranslateGeometry(float64(mx), float64(my)))
		clr := ebiten.ColorMatrixI()
		clr.Concat(ebiten.ScaleColor(color.RGBA{0xff, 0xff, 0x00, 0xff}))
		theta := 2.0 * math.Pi * float64(g.count%60) / 60.0
		clr.Concat(ebiten.RotateHue(theta))
		ebiten.DrawWhole(gr.RenderTarget(g.brushRenderTarget), 1, 1, geo, clr)
		gr.PopRenderTarget()
	}

	ebiten.DrawWhole(gr.RenderTarget(g.canvasRenderTarget), screenWidth, screenHeight, ebiten.GeometryMatrixI(), ebiten.ColorMatrixI())

	ebitenutil.DebugPrint(gr, fmt.Sprintf("(%d, %d)", mx, my))
	return nil
}

func init() {
	runtime.LockOSThread()
}

func main() {
	game := new(Game)
	if err := ebiten.Run(game, screenWidth, screenHeight, 2, "Paint (Ebiten Demo)", 60); err != nil {
		log.Fatal(err)
	}
}
