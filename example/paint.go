package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/runner"
	"log"
	"runtime"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	gameContext        ebiten.GameContext
	canvasRenderTarget ebiten.RenderTargetID
}

func (g *Game) Initialize(ga ebiten.GameContext) error {
	g.gameContext = ga
	return nil
}

func (g *Game) Update() error {
	// TODO: Implement
	return nil
}

func (g *Game) Draw(gr ebiten.GraphicsContext) error {
	if g.canvasRenderTarget.IsNil() {
		var err error
		g.canvasRenderTarget, err = g.gameContext.NewRenderTargetID(screenWidth, screenHeight, ebiten.FilterNearest)
		if err != nil {
			return err
		}
		gr.SetOffscreen(g.canvasRenderTarget)
		gr.Fill(0xff, 0xff, 0xff)
	}
	gr.ResetOffscreen()
	ebiten.DrawWhole(gr.RenderTarget(g.canvasRenderTarget), screenWidth, screenHeight, ebiten.GeometryMatrixI(), ebiten.ColorMatrixI())
	ebitenutil.DebugPrint(g.gameContext, gr, "Hello\nWorld!")
	return nil
}

func init() {
	runtime.LockOSThread()
}

func main() {
	game := new(Game)
	if err := runner.Run(game, screenWidth, screenHeight, 2, "Paint (Ebiten Demo)", 60); err != nil {
		log.Fatal(err)
	}
}
