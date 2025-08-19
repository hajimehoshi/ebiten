package main

import (
    "os"
    "image"

    "github.com/hajimehoshi/ebiten/v2"
)

func loadImage(path string) (*ebiten.Image, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    img, _, err := image.Decode(file)
    if err != nil {
        return nil, err
    }
    return ebiten.NewImageFromImage(img), nil
}

func drawBackground(screen *ebiten.Image, bg *ebiten.Image) {
    op := &ebiten.DrawImageOptions{}
    screen.DrawImage(bg, op)
}
