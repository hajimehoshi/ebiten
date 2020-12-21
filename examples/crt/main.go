// Copyright 2020 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build example

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	images "github.com/hajimehoshi/ebiten/v2/examples/resources/images/crt"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// Game implements ebiten.Game interface.
type Game struct {
	Objects   []*Object
	Offscreen *ebiten.Image
	Time      float32
	Score     int
}

type Name int

const (
	Player Name = iota
	Ebiten
	Floor
)

// Object implements an object that the game loop modifies and the draw loop renders
type Object struct {
	Name          Name
	X, Y, Z, W, H int
	VX, VY        float64
	Flipped       bool
	Image         *ebiten.Image
	Tile          bool
	TileX         int
	TileY         int
	Hidden        bool
}

// Draw executes graphical operations to draw a scene
func (o *Object) Draw(screen *ebiten.Image, OffsetX, OffsetY float64) {
	if !o.Tile {
		opts := &ebiten.DrawImageOptions{}
		if o.Flipped {
			opts.GeoM.Scale(-1, 1)
			opts.GeoM.Translate(float64(o.W), 0)
		}
		opts.GeoM.Translate(OffsetX+float64(o.X), OffsetY+float64(o.Y))
		screen.DrawImage(o.Image, opts)
	} else {
		for x := 0; x < o.TileX; x++ {
			for y := 0; y < o.TileY; y++ {
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(OffsetX+float64(o.X)+float64(x*o.W), OffsetY+float64(o.Y)+float64(y*o.H))
				screen.DrawImage(o.Image, opts)
			}
		}
	}
}

func getPlayerIndex(objs []*Object) int {
	for i, obj := range objs {
		if obj.Name == Player {
			return i
		}
	}
	return 0
}

func handlePlayerInput(objs []*Object) {
	i := getPlayerIndex(objs)
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		objs[i].VX -= 2
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		objs[i].VX += 2
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		objs[i].VY -= 80
	}
}

func (g *Game) ebitenSpawner() {
	t := int(g.Time) % 60
	if t == 0 {
		g.Objects = append(g.Objects, &Object{
			Name:  Ebiten,
			X:     0,
			Y:     rand.Intn(160 + 1),
			Z:     1,
			W:     57,
			H:     26,
			VX:    float64(rand.Intn(150-10+1) + 10),
			VY:    0,
			Image: ebitenImage,
		})
	}
}

func (g *Game) ebitenCollisionDetect(o *Object) bool {
	aw, ah := ebitenImage.Size()
	gaw, gah := gopherImage.Size()
	i := getPlayerIndex(g.Objects)
	p := g.Objects[i]
	if p.X < o.X+aw &&
		p.X+gaw > o.X &&
		p.Y < o.Y+ah &&
		p.Y+gah > o.Y {
		return true
	}
	return false
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update() error {
	g.Time++
	handlePlayerInput(g.Objects)
	g.ebitenSpawner()
	for _, obj := range g.Objects {
		if obj.Hidden {
			continue
		} else {
			if obj.Name == Ebiten {
				if g.ebitenCollisionDetect(obj) {
					obj.Hidden = true
					g.Score++
				}
			}
			if obj.Name == Player {
				if (obj.Y + obj.H) < 320 {
					obj.VY += 8
				}
				if (obj.Y + obj.H) > 320 {
					obj.Y = 319 - obj.H
					obj.VY = 1 //subtle bounce effect
				}
				if obj.VX > 0 {
					obj.Flipped = false
				} else if obj.VX < 0 {
					obj.Flipped = true
				}
			}
			obj.X += int(obj.VX)
			obj.VX /= 8
			obj.Y += int(obj.VY)
			obj.VY /= 8
		}
	}
	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	// Write your game's rendering.
	bgopts := &ebiten.DrawImageOptions{}
	g.Offscreen.DrawImage(bgImage, bgopts)
	i := getPlayerIndex(g.Objects)
	p := g.Objects[i]
	drawOffsetX := -(p.VX / 640) * 200
	drawOffsetY := -(p.VY / 480) * 50
	for z := 0; z < 3; z++ {
		for _, obj := range g.Objects {
			if !obj.Hidden {
				if obj.Z == z {
					obj.Draw(g.Offscreen, drawOffsetX, drawOffsetY)
				}
			}
		}
	}
	xJitter, yJitter := math.Sincos(float64(g.Time / 10))
	text.Draw(g.Offscreen, "COLLECT EBITEN!", mainFont, int(drawOffsetX+340+xJitter*2), int(drawOffsetY+40+yJitter*2), color.White)
	text.Draw(g.Offscreen, fmt.Sprintf("%d", g.Score), mainFont, int(drawOffsetX+50+xJitter*2), int(drawOffsetY+40+yJitter*2), color.White)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(drawOffsetX+xJitter*2, drawOffsetY+yJitter*2)
	g.Offscreen.DrawImage(ebitenImage, opts)
	// These last few lines of the draw loop are of concern if you are looking to execute the crt shader.
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"Time": float32(g.Time) / 60,
	}
	op.Images[0] = g.Offscreen
	screen.DrawRectShader(640, 480, ntscShader, op)
	g.Offscreen.Clear()
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

var (
	gopherImage *ebiten.Image
	floorImage  *ebiten.Image
	ebitenImage *ebiten.Image
	bgImage     *ebiten.Image
	ntscShader  *ebiten.Shader
	mainFont    font.Face
)

func main() {
	rand.Seed(time.Now().UnixNano())
	tt, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72
	mainFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    20,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	imgtemp, _, err := image.Decode(bytes.NewReader(images.Gopher_png))
	if err != nil {
		log.Fatal(err)
	}
	gopherImage = ebiten.NewImageFromImage(imgtemp)
	imgtemp, _, err = image.Decode(bytes.NewReader(images.Floor_png))
	if err != nil {
		log.Fatal(err)
	}
	floorImage = ebiten.NewImageFromImage(imgtemp)
	imgtemp, _, err = image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(imgtemp)
	imgtemp, _, err = image.Decode(bytes.NewReader(images.Bg_png))
	if err != nil {
		log.Fatal(err)
	}
	bgImage = ebiten.NewImageFromImage(imgtemp)
	objs := make([]*Object, 0)
	objs = append(objs, &Object{
		Name:  Player,
		X:     160,
		Y:     80,
		Z:     1,
		W:     60,
		H:     75,
		Image: gopherImage,
	})
	objs = append(objs, &Object{
		Name:  Floor,
		X:     -64,
		Y:     320,
		Z:     1,
		W:     64,
		H:     64,
		Tile:  true,
		TileX: 13,
		TileY: 3,
		Image: floorImage,
	})
	offscreen := ebiten.NewImage(640, 480)
	game := &Game{
		Objects:   objs,
		Offscreen: offscreen,
	}
	s, err := ebiten.NewShader(ntsc_go)
	if err != nil {
		log.Fatal(err)
	}
	ntscShader = s
	// Specify the window size as you like. Here, a doubled size is specified.
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("CRT Shader Example")
	// Call ebiten.RunGame to start your game loop.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
