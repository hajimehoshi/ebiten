package main

import (
    "image/gif"
    "os"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
)

type GIFAnimator struct {
    Frames       []*ebiten.Image
    FrameDelays  []int // in 10ms units
    currentFrame int
    lastTime     time.Time
}

func NewGIFAnimator(gifPath string) (*GIFAnimator, error) {
    f, err := os.Open(gifPath)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    g, err := gif.DecodeAll(f)
    if err != nil {
        return nil, err
    }

    frames := make([]*ebiten.Image, len(g.Image))
    for i, img := range g.Image {
        ebitenImg := ebiten.NewImageFromImage(img)
        frames[i] = ebitenImg
    }

    return &GIFAnimator{
        Frames:       frames,
        FrameDelays:  g.Delay,
        currentFrame: 0,
        lastTime:     time.Now(),
    }, nil
}

func (a *GIFAnimator) Update() {
    now := time.Now()
    elapsed := now.Sub(a.lastTime)
    delay := a.FrameDelays[a.currentFrame] * 10 // convert to milliseconds

    if elapsed >= time.Duration(delay)*time.Millisecond {
        a.currentFrame = (a.currentFrame + 1) % len(a.Frames)
        a.lastTime = now
    }
}

func (a *GIFAnimator) Draw(screen *ebiten.Image, x, y float64) {
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(x, y)
    screen.DrawImage(a.Frames[a.currentFrame], op)
}
