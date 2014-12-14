package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"log"
	"runtime"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	canvasRenderTarget ebiten.RenderTargetID
}

func (g *Game) Update() error {
	// TODO: Implement
	return nil
}

func (g *Game) Draw(gr ebiten.GraphicsContext) error {
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
	ebiten.DrawWhole(gr.RenderTarget(g.canvasRenderTarget), screenWidth, screenHeight, ebiten.GeometryMatrixI(), ebiten.ColorMatrixI())

	mx, my := ebiten.CursorPosition()
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
