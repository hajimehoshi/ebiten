// Copyright 2024 The Ebitengine Authors
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

package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	resources "github.com/hajimehoshi/ebiten/v2/examples/resources/images/shader"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 1920
	screenHeight = 1080
)

var (
	dsts = [8]*ebiten.Image{
		ebiten.NewImageWithOptions(image.Rect(0, 0, screenWidth, screenHeight), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
		ebiten.NewImageWithOptions(image.Rect(0, 0, screenWidth, screenHeight), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
	}

	shaderSpriteSrc = []byte(
		`
//kage:units pixels

package main

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	return imageSrc0UnsafeAt(src)
}
	`)

	shaderSpriteMRTSrc = []byte(
		`
//kage:units pixels

package main

func Fragment(dst vec4, src vec2, color vec4) (vec4, vec4, vec4) {
	return imageSrc0UnsafeAt(src), imageSrc1UnsafeAt(src), imageSrc2UnsafeAt(src)
}
`)

	shaderFinalSrc = []byte(
		`
// kage:unit pixels

package main

var Cursor vec2

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	pos := dst.xy - imageDstOrigin()

	lightpos := vec3(Cursor, 150)
	lightdir := normalize(lightpos - vec3(pos, 0))
	normal := normalize(imageSrc1UnsafeAt(src) - 0.5).xyz
	clr := imageSrc0UnsafeAt(src)
	ambient := 0.1*clr.rgb
	diffuse := 0.9*clr.rgb * max(0.0, dot(normal, lightdir))
	reflectDir := reflect(-lightdir, normal)
    spec := 0.25*pow(max(1-dot(reflectDir, reflectDir), 0.0), imageSrc2UnsafeAt(src).x*16)

	return vec4(vec3(ambient + diffuse + spec), 1)*clr.a
}
`)

	spriteShader    *ebiten.Shader
	spriteMRTShader *ebiten.Shader
	finalShader     *ebiten.Shader
)

var (
	gopherImage   *ebiten.Image
	normalImage   *ebiten.Image
	specularImage *ebiten.Image
)

func init() {
	var err error

	spriteShader, err = ebiten.NewShader(shaderSpriteSrc)
	if err != nil {
		log.Fatal(err)
	}
	spriteMRTShader, err = ebiten.NewShader(shaderSpriteMRTSrc)
	if err != nil {
		log.Fatal(err)
	}
	finalShader, err = ebiten.NewShader(shaderFinalSrc)
	if err != nil {
		log.Fatal(err)
	}

	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(resources.Gopher_png))
	if err != nil {
		log.Fatal(err)
	}
	gopherImage = ebiten.NewImageFromImage(img)

	img, _, err = image.Decode(bytes.NewReader(resources.Normal_png))
	if err != nil {
		log.Fatal(err)
	}
	tmpNormal := ebiten.NewImageFromImage(img)

	img, _, err = image.Decode(bytes.NewReader(resources.Specular_png))
	if err != nil {
		log.Fatal(err)
	}
	specularImage = ebiten.NewImageFromImage(img)

	// Fixing the normal background
	normalImage = ebiten.NewImage(tmpNormal.Bounds().Dx(), tmpNormal.Bounds().Dy())
	normalImage.DrawImage(gopherImage, nil)
	normalImage.DrawImage(tmpNormal, &ebiten.DrawImageOptions{
		Blend: ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorDestinationAlpha,
			BlendFactorSourceAlpha:      ebiten.BlendFactorDestinationAlpha,
			BlendFactorDestinationRGB:   ebiten.BlendFactorDefault,
			BlendFactorDestinationAlpha: ebiten.BlendFactorDefault,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		},
	})
}

type Game struct {
	mrt      bool
	index    int
	vertices []ebiten.Vertex
	indices  []uint16

	color    *ebiten.Image
	normal   *ebiten.Image
	specular *ebiten.Image
}

var quadIndices = []uint16{0, 1, 2, 1, 2, 3}

func (g *Game) AddSprite() {
	x := rand.Float32() * screenWidth
	y := rand.Float32() * screenHeight
	w := (0.5 + rand.Float32()*0.5) * float32(gopherImage.Bounds().Dx())
	h := (0.5 + rand.Float32()*0.5) * float32(gopherImage.Bounds().Dy())
	g.vertices = append(g.vertices,
		ebiten.Vertex{
			DstX: x,
			DstY: y,
		},
		ebiten.Vertex{
			DstX: x + w,
			DstY: y,
			SrcX: float32(gopherImage.Bounds().Dx()),
		},
		ebiten.Vertex{
			DstX: x,
			DstY: y + h,
			SrcY: float32(gopherImage.Bounds().Dy()),
		},
		ebiten.Vertex{
			DstX: x + w,
			DstY: y + h,
			SrcX: float32(gopherImage.Bounds().Dx()),
			SrcY: float32(gopherImage.Bounds().Dy()),
		},
	)
	index := uint16(g.index * 4)
	g.indices = append(g.indices,
		quadIndices[0]+index,
		quadIndices[1]+index,
		quadIndices[2]+index,
		quadIndices[3]+index,
		quadIndices[4]+index,
		quadIndices[5]+index,
	)

	g.index++
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		for i := 0; i < 20; i++ {
			g.AddSprite()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.mrt = !g.mrt
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.color.Clear()
	g.normal.Clear()
	g.specular.Clear()

	if g.mrt {
		ebiten.DrawTrianglesShaderMRT([8]*ebiten.Image{
			g.color, g.normal, g.specular,
		}, g.vertices, g.indices, spriteMRTShader, &ebiten.DrawTrianglesShaderOptions{
			Images: [4]*ebiten.Image{
				gopherImage, normalImage, specularImage,
			},
		})
	} else {
		g.color.DrawTrianglesShader(g.vertices, g.indices, spriteShader, &ebiten.DrawTrianglesShaderOptions{
			Images: [4]*ebiten.Image{
				gopherImage,
			},
		})
		// Note: batch broken here since a different unmanaged image could be bound, so unfair (both these calls could be batched)
		g.normal.DrawTrianglesShader(g.vertices, g.indices, spriteShader, &ebiten.DrawTrianglesShaderOptions{
			Images: [4]*ebiten.Image{
				normalImage,
			},
		})
		// Note: Here too
		g.specular.DrawTrianglesShader(g.vertices, g.indices, spriteShader, &ebiten.DrawTrianglesShaderOptions{
			Images: [4]*ebiten.Image{
				specularImage,
			},
		})
	}

	cx, cy := ebiten.CursorPosition()
	screen.DrawTrianglesShader([]ebiten.Vertex{
		{},
		{
			DstX: screenWidth,
			SrcX: screenWidth,
		},
		{
			DstY: screenHeight,
			SrcY: screenHeight,
		},
		{
			DstX: screenWidth,
			DstY: screenHeight,
			SrcX: screenWidth,
			SrcY: screenHeight,
		},
	}, quadIndices, finalShader, &ebiten.DrawTrianglesShaderOptions{
		Images: [4]*ebiten.Image{
			g.color, g.normal, g.specular,
		},
		Uniforms: map[string]any{
			"Cursor": []float32{
				float32(cx), float32(cy),
			},
		},
	})

	ebitenutil.DebugPrint(screen, fmt.Sprintf("MRT: %v - Count: %d - FPS: %.2f", g.mrt, g.index, ebiten.ActualFPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("MRT Sprites (Ebitengine Demo)")

	if err := ebiten.RunGameWithOptions(&Game{
		color: ebiten.NewImageWithOptions(
			image.Rect(0, 0, screenWidth, screenHeight),
			&ebiten.NewImageOptions{
				Unmanaged: true,
			},
		),
		normal: ebiten.NewImageWithOptions(
			image.Rect(0, 0, screenWidth, screenHeight),
			&ebiten.NewImageOptions{
				Unmanaged: true,
			},
		),
		specular: ebiten.NewImageWithOptions(
			image.Rect(0, 0, screenWidth, screenHeight),
			&ebiten.NewImageOptions{
				Unmanaged: true,
			},
		),
	}, &ebiten.RunGameOptions{
		GraphicsLibrary: ebiten.GraphicsLibraryOpenGL,
	}); err != nil {
		log.Fatal(err)
	}
}
