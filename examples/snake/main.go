package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	xNumInScreen = screenWidth / gridSize
	yNumInScreen = screenHeight / gridSize
	gridSize     = 10
)

const (
	dirLeft = iota + 1
	dirRight
	dirDown
	dirUp
)

type Position struct {
	X int64
	Y int64
}

type Game struct {
	moveDirection int8
	appleX        int64
	appleY        int64
	snakeBody     []Position
	timer         uint64
	speed         int8
	score         int64
	bestScore     int64
	level         int8
}

func (g *Game) collidesWithApple() bool {
	return g.snakeBody[0].X == g.appleX &&
		g.snakeBody[0].Y == g.appleY
}

func (g *Game) collidesWithSelf() bool {
	for _, v := range g.snakeBody[1:] {
		if g.snakeBody[0].X == v.X &&
			g.snakeBody[0].Y == v.Y {
			return true
		}
	}
	return false
}

func (g *Game) collidesWithWall() bool {
	return g.snakeBody[0].X < 0 ||
		g.snakeBody[0].Y < 0 ||
		g.snakeBody[0].X >= xNumInScreen ||
		g.snakeBody[0].Y >= yNumInScreen
}

func (g *Game) updateTimer() {
	g.timer++
}

func (g *Game) needsToMoveSnake() bool {
	return g.timer%uint64(g.speed) == 0
}

func (g *Game) reset() {
	g.appleX = 3 * gridSize
	g.appleY = 3 * gridSize
	g.speed = 4
	g.snakeBody = g.snakeBody[:1]
	g.snakeBody[0].X = xNumInScreen / 2
	g.snakeBody[0].Y = yNumInScreen / 2
	g.score = 0
	g.level = 1
	g.moveDirection = 0
}

func (g *Game) Update(screen *ebiten.Image) error {

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		if g.moveDirection != dirRight {
			g.moveDirection = dirLeft
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if g.moveDirection != dirLeft {
			g.moveDirection = dirRight
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if g.moveDirection != dirUp {
			g.moveDirection = dirDown
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if g.moveDirection != dirDown {
			g.moveDirection = dirUp
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.reset()
	}

	if g.needsToMoveSnake() {
		if g.collidesWithWall() || g.collidesWithSelf() {
			g.reset()
		}

		if g.collidesWithApple() {
			g.appleX = rand.Int63n(xNumInScreen - 1)
			g.appleY = rand.Int63n(yNumInScreen - 1)
			g.snakeBody = append(g.snakeBody, Position{
				X: g.snakeBody[int64(len(g.snakeBody)-1)].X,
				Y: g.snakeBody[int64(len(g.snakeBody)-1)].Y,
			})
			if len(g.snakeBody) > 10 && len(g.snakeBody) < 20 {
				g.level = 2
				g.speed = 3
			} else if len(g.snakeBody) > 20 {
				g.level = 3
				g.speed = 2
			} else {
				g.level = 1
			}
			g.score++
			if g.bestScore < g.score {
				g.bestScore = g.score
			}
		}

		for i := int64(len(g.snakeBody)) - 1; i > 0; i-- {
			g.snakeBody[i].X = g.snakeBody[i-1].X
			g.snakeBody[i].Y = g.snakeBody[i-1].Y
		}
		switch g.moveDirection {
		case dirLeft:
			g.snakeBody[0].X--
		case dirRight:
			g.snakeBody[0].X++
		case dirDown:
			g.snakeBody[0].Y++
		case dirUp:
			g.snakeBody[0].Y--
		}
	}

	g.updateTimer()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.moveDirection == 0 {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("Press up/down/left/right to start"))
	} else {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f Level: %d Score: %d Best Score: %d", ebiten.CurrentFPS(), g.level, g.score, g.bestScore))
	}

	for _, v := range g.snakeBody {
		ebitenutil.DrawRect(screen, float64(v.X*gridSize), float64(v.Y*gridSize), gridSize, gridSize, color.RGBA{0x80, 0xa0, 0xc0, 0xff})
	}
	ebitenutil.DrawRect(screen, float64(g.appleX*gridSize), float64(g.appleY*gridSize), gridSize, gridSize, color.RGBA{0xFF, 0x00, 0x00, 0xff})
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func newGame() *Game {
	g := &Game{
		appleX:    3 * gridSize,
		appleY:    3 * gridSize,
		speed:     4,
		snakeBody: make([]Position, 1),
	}
	g.snakeBody[0].X = xNumInScreen / 2
	g.snakeBody[0].Y = yNumInScreen / 2
	return g
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Snake (Ebiten Demo)")
	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
