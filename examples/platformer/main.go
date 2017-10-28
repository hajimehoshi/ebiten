package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	// Settings
	width           int  = 1024
	height          int  = 512
	fullscreen      bool = false
	runinbackground bool = true
)

var (
	LoadedSprite    *ebiten.Image
	LeftSprite      *ebiten.Image
	RightSprite     *ebiten.Image
	IdleSprite      *ebiten.Image
	BackgroundImage *ebiten.Image

	IsFirstFrame bool = true

	BackgroundOptions = &ebiten.DrawImageOptions{}
	CharSpriteOptions = &ebiten.DrawImageOptions{}
)

func update(screen *ebiten.Image) error {
	// Draws Background Image
	screen.DrawImage(BackgroundImage, BackgroundOptions)

	// Resets
	charX := 0.0
	charY := 0.0

	// I'm doing this to easily define a starting position for the gopher
	if IsFirstFrame == true {
		charY = charY + 390
		charX = charX + 50
		IsFirstFrame = false
	}

	// Controls
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		// Selects preloaded sprite
		LoadedSprite = LeftSprite
		// Moves character 3px right
		charX = charX - 3.0
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		// Selects preloaded sprite
		LoadedSprite = RightSprite
		// Moves character 3px left
		charX = charX + 3.0
	} else {
		LoadedSprite = IdleSprite
	}

	// Change gopher's position
	CharSpriteOptions.GeoM.Translate(charX, charY)

	// FPS counter
	fps := fmt.Sprintf("FPS: %f", ebiten.CurrentFPS())
	ebitenutil.DebugPrint(screen, fps)

	// Draws selected sprite image
	screen.DrawImage(LoadedSprite, CharSpriteOptions)

	return nil
}

func main() {
	// Settings for images
	BackgroundOptions.GeoM.Scale(0.5, 0.5)
	BackgroundOptions.GeoM.Translate(0, 0)
	CharSpriteOptions.GeoM.Scale(0.5, 0.5)

	// Preload images, This isn't proper error handling however has been kept short for simplicity purposes
	RightSprite, _, _ = ebitenutil.NewImageFromFile("img/right.png", ebiten.FilterNearest)
	LeftSprite, _, _ = ebitenutil.NewImageFromFile("img/left.png", ebiten.FilterNearest)
	IdleSprite, _, _ = ebitenutil.NewImageFromFile("img/mainchar.png", ebiten.FilterNearest)
	BackgroundImage, _, _ = ebitenutil.NewImageFromFile("img/background.png", ebiten.FilterNearest)

	ebiten.SetRunnableInBackground(runinbackground)
	ebiten.SetFullscreen(fullscreen)

	// Starts the program
	ebiten.Run(update, width, height, 1, "Platformer")
}
