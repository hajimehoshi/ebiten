// Copyright 2021 The Ebiten Authors
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

package atlas_test

import (
	"image"
	"image/color"
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/legacyshader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestShaderFillTwice(t *testing.T) {
	const w, h = 1, 1

	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	g := ui.Get().GraphicsDriverForTesting()
	s0 := atlas.NewShader(etesting.ShaderProgramFill(0xff, 0xff, 0xff, 0xff), "")
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, s0, nil)

	// Vertices must be recreated (#1755)
	vs = quadVertices(w, h, 0, 0, 1)
	s1 := atlas.NewShader(etesting.ShaderProgramFill(0x80, 0x80, 0x80, 0xff), "")
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, s1, nil)

	pix := make([]byte, 4*w*h)
	ok, err := dst.ReadPixels(g, pix, image.Rect(0, 0, w, h))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Fatal("ReadPixels failed")
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageDrawTwice(t *testing.T) {
	const w, h = 1, 1

	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src0 := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src0.WritePixels([]byte{0xff, 0xff, 0xff, 0xff}, image.Rect(0, 0, w, h))
	src1 := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src1.WritePixels([]byte{0x80, 0x80, 0x80, 0xff}, image.Rect(0, 0, w, h))

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	sr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*atlas.Image{src0}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, atlas.NearestFilterShader, nil)

	// Vertices must be recreated (#1755)
	vs = quadVertices(w, h, 0, 0, 1)
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*atlas.Image{src1}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, atlas.NearestFilterShader, nil)

	pix := make([]byte, 4*w*h)
	ok, err := dst.ReadPixels(ui.Get().GraphicsDriverForTesting(), pix, image.Rect(0, 0, w, h))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Fatal("ReadPixels failed")
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

type shaderDisposalObserver struct {
	graphicsdriver.Graphics
	program  *shaderir.Program
	created  chan struct{}
	disposed chan struct{}
}

func (g *shaderDisposalObserver) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	shader, err := g.Graphics.NewShader(program)
	if err != nil {
		return nil, err
	}
	if program != g.program {
		return shader, nil
	}
	close(g.created)
	return &observedShader{
		Shader:   shader,
		disposed: g.disposed,
	}, nil
}

type observedShader struct {
	graphicsdriver.Shader
	disposed chan struct{}
}

func (s *observedShader) Dispose() {
	s.Shader.Dispose()
	close(s.disposed)
}

func TestGCShader(t *testing.T) {
	program := etesting.ShaderProgramFill(0xff, 0xff, 0xff, 0xff)
	s := atlas.NewShader(program, "")
	g := &shaderDisposalObserver{
		Graphics: ui.Get().GraphicsDriverForTesting(),
		program:  program,
		created:  make(chan struct{}),
		disposed: make(chan struct{}),
	}

	// Use the shader to initialize it.
	const w, h = 1, 1
	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	defer dst.Deallocate()
	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, s, nil)
	if err := graphicscommand.FlushCommands(g, graphicsdriver.FlushModeIntermediate); err != nil {
		t.Fatal(err)
	}
	select {
	case <-g.created:
	default:
		t.Fatal("shader was not created")
	}
	runtime.KeepAlive(s)

	if !waitForGC(func() bool {
		if err := graphicscommand.FlushCommands(g, graphicsdriver.FlushModeIntermediate); err != nil {
			t.Fatal(err)
		}
		select {
		case <-g.disposed:
			return true
		default:
			return false
		}
	}) {
		t.Error("shader was not disposed after GC")
	}
}

func TestGCShaderRemovesRegistryEntry(t *testing.T) {
	const w, h = 1, 1
	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	defer dst.Deallocate()
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)

	const count = 10
	shaders := make([]*atlas.Shader, 0, count)
	checks := make([]func() bool, 0, count)
	for range count {
		s := atlas.NewShader(etesting.ShaderProgramFill(0xff, 0xff, 0xff, 0xff), "")
		// Use the shader to initialize its internal shader.
		vs := quadVertices(w, h, 0, 0, 1)
		dst.DrawTriangles([graphics.ShaderSrcImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, s, nil)
		shaders = append(shaders, s)
		checks = append(checks, s.IsRegisteredFuncForTesting())
	}
	for i, registered := range checks {
		if !registered() {
			t.Fatalf("shader %d was not registered", i)
		}
	}
	runtime.KeepAlive(shaders)
	shaders = nil

	if !waitForGC(func() bool {
		for _, registered := range checks {
			if registered() {
				return false
			}
		}
		return true
	}) {
		for i, registered := range checks {
			if registered() {
				t.Errorf("shader %d remained registered after GC", i)
			}
		}
	}
}

func TestBuiltinShaderSourceIDs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		src    []byte
		shader *atlas.Shader
	}{
		{
			name:   "nearest",
			src:    builtinshader.ShaderSource(builtinshader.FilterNearest, builtinshader.AddressUnsafe),
			shader: atlas.NearestFilterShader,
		},
		{
			name:   "linear",
			src:    builtinshader.ShaderSource(builtinshader.FilterLinear, builtinshader.AddressUnsafe),
			shader: atlas.LinearFilterShader,
		},
		{
			name:   "clear",
			src:    []byte(builtinshader.ClearShaderSource),
			shader: atlas.ClearShaderForTesting(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, err := legacyshader.CalcSourceID(tc.src)
			if err != nil {
				t.Fatal(err)
			}
			if got := tc.shader.SourceIDForTesting(); got != want {
				t.Errorf("source ID: got: %s, want: %s", got, want)
			}
		})
	}
}
