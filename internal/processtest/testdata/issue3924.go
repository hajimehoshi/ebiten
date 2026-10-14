// Copyright 2026 The Ebitengine Authors
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

//go:build ignore

package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// This test confirms that a shader drawn on the final screen observes the same dstPos, dfdy, and frontfacing
// as on an offscreen image with OpenGL, whose default framebuffer's window coordinates start at the bottom (#3924).

// observations are what the shader observes, in the order of the bits of the red channel.
var observations = []string{
	"dstPos.y < imageDstSize().y/2",
	"fract(dstPos.y) == 0.5",
	"dfdy(dstPos.y) > 0",
	"dfdy(src0Pos.y) > 0",
	"frontfacing()",
}

const shaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	bits := 0.0
	if dstPos.y-imageDstOrigin().y < imageDstSize().y/2 {
		bits += 1
	}
	if fract(dstPos.y) == 0.5 {
		bits += 2
	}
	if dfdy(dstPos.y) > 0 {
		bits += 4
	}
	if dfdy(src0Pos.y) > 0 {
		bits += 8
	}
	if frontfacing() {
		bits += 16
	}
	return vec4(bits/255, 0, 0, 1)
}
`

type game struct {
	shader *ebiten.Shader
	done   bool
	err    error
}

func (g *game) Update() error {
	if g.done {
		if g.err != nil {
			return g.err
		}
		return ebiten.Termination
	}
	if g.shader == nil {
		s, err := ebiten.NewShader([]byte(shaderSrc))
		if err != nil {
			return err
		}
		g.shader = s
	}
	return nil
}

func (*game) Draw(*ebiten.Image) {
}

// rectTriangles returns two triangles with the same winding that cover the rectangle from (x, 0) to (x+width, height).
// The source positions are relative to the rectangle.
func rectTriangles(x, width, height float32) ([]ebiten.Vertex, []uint16) {
	vertices := []ebiten.Vertex{
		{
			DstX: x,
			DstY: 0,
			SrcX: 0,
			SrcY: 0,
		},
		{
			DstX: x + width,
			DstY: 0,
			SrcX: width,
			SrcY: 0,
		},
		{
			DstX: x,
			DstY: height,
			SrcX: 0,
			SrcY: height,
		},
		{
			DstX: x + width,
			DstY: height,
			SrcX: width,
			SrcY: height,
		},
	}
	return vertices, []uint16{0, 1, 2, 2, 1, 3}
}

func (g *game) DrawFinalScreen(screen ebiten.FinalScreen, _ *ebiten.Image, _ ebiten.GeoM) {
	if g.done || g.shader == nil {
		return
	}

	// The left half is the shader's result on an offscreen image, and the right half is the shader's result on the final screen.
	// The final screen's size depends on the device scale factor.
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w := sw / 2
	if w == 0 || sh < 2 {
		return
	}

	offscreen := ebiten.NewImage(w, sh)
	vs, is := rectTriangles(0, float32(w), float32(sh))
	offscreen.DrawTrianglesShader(vs, is, g.shader, nil)
	screen.DrawImage(offscreen, nil)

	vs, is = rectTriangles(float32(w), float32(w), float32(sh))
	screen.DrawTrianglesShader(vs, is, g.shader, nil)

	pix := make([]byte, 4*sw*sh)
	screen.(*ebiten.Image).ReadPixels(pix)
	g.err = compareHalves(pix, sw, sh, w)
	g.done = true
}

// compareHalves compares the left half with the right half of the final screen's pixels row by row.
// The rows' order does not matter, as the final screen's pixels might be read from the bottom.
func compareHalves(pix []byte, screenWidth, screenHeight, halfWidth int) error {
	var errs []error

	var top, bottom bool
	mismatches := make([]int, len(observations))
	for j := range screenHeight {
		for i := range halfWidth {
			want := pix[4*(j*screenWidth+i) : 4*(j*screenWidth+i)+4]
			got := pix[4*(j*screenWidth+halfWidth+i) : 4*(j*screenWidth+halfWidth+i)+4]
			if want[3] != 0xff || got[3] != 0xff {
				return fmt.Errorf("pixels at row %d, column %d are not drawn: offscreen: %v, final screen: %v", j, i, want, got)
			}
			if want[0]&1 != 0 {
				top = true
			} else {
				bottom = true
			}
			for bit := range observations {
				if (want[0]^got[0])&(1<<bit) != 0 {
					mismatches[bit]++
				}
			}
		}
	}
	if !top || !bottom {
		errs = append(errs, errors.New("the offscreen image must have both the top and the bottom halves"))
	}
	for bit, n := range mismatches {
		if n > 0 {
			errs = append(errs, fmt.Errorf("%s differs between an offscreen image and the final screen at %d of %d pixels", observations[bit], n, halfWidth*screenHeight))
		}
	}
	return errors.Join(errs...)
}

func (*game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	// OpenGL might not be available on GitHub Actions' Windows runners.
	if runtime.GOOS == "windows" && strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")) == "true" {
		return
	}

	ebiten.SetWindowVisible(false)
	// In float32, 2/(2/height) is not exactly the height for this height or twice it (a device scale factor of 2).
	ebiten.SetWindowSize(64, 30)
	if err := ebiten.RunGameWithOptions(&game{}, &ebiten.RunGameOptions{
		GraphicsLibrary: ebiten.GraphicsLibraryOpenGL,
	}); err != nil {
		panic(err)
	}
}
