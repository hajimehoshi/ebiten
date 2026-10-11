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

package directx_test

import (
	"bytes"
	"image"
	"os"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/directx"
)

func TestReadPixelsAsync(t *testing.T) {
	for _, version := range []string{"version=11", "version=12"} {
		t.Run(version, func(t *testing.T) {
			t.Setenv("EBITENGINE_DIRECTX", os.Getenv("EBITENGINE_DIRECTX")+","+version)
			g, err := directx.NewGraphics()
			if err != nil {
				t.Fatal(err)
			}
			if err := g.Initialize(); err != nil {
				t.Fatal(err)
			}
			if err := g.Begin(); err != nil {
				t.Fatal(err)
			}
			img, err := g.NewImage(129, 67)
			if err != nil {
				t.Fatal(err)
			}
			reader, ok := img.(graphicsdriver.AsyncPixelsReader)
			if !ok {
				t.Fatal("DirectX does not implement AsyncPixelsReader")
			}
			source := make([]byte, 129*67*4)
			for y := 0; y < 67; y++ {
				for x := 0; x < 129; x++ {
					copy(source[4*(y*129+x):], []byte{byte(x), byte(y), byte(x + y), 255})
				}
			}
			if err := img.WritePixels([]graphicsdriver.PixelsArgs{{Pixels: source, Region: image.Rect(0, 0, 129, 67)}}); err != nil {
				t.Fatal(err)
			}
			regions := []image.Rectangle{image.Rect(3, 5, 10, 16), image.Rect(99, 61, 129, 67), image.Rect(0, 0, 1, 1)}
			args := make([]graphicsdriver.PixelsArgs, len(regions))
			for j, r := range regions {
				args[j] = graphicsdriver.PixelsArgs{Region: r, Pixels: make([]byte, 4*r.Dx()*r.Dy())}
			}
			readback, err := reader.ReadPixelsAsync(args)
			if err != nil {
				t.Fatal(err)
			}
			defer readback.Dispose()
			for j := range source {
				source[j] = 0
			}
			if err := img.WritePixels([]graphicsdriver.PixelsArgs{{Pixels: source, Region: image.Rect(0, 0, 129, 67)}}); err != nil {
				t.Fatal(err)
			}
			secondArgs := []graphicsdriver.PixelsArgs{{Region: image.Rect(17, 19, 26, 24), Pixels: make([]byte, 4*9*5)}}
			second, err := reader.ReadPixelsAsync(secondArgs)
			if err != nil {
				t.Fatal(err)
			}
			defer second.Dispose()
			img.Dispose()
			deadline := time.Now().Add(5 * time.Second)
			for _, r := range []graphicsdriver.PixelsReadback{readback, second} {
				for {
					done, err := r.Poll()
					if err != nil {
						t.Fatal(err)
					}
					if done {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("read-back did not complete without a later flush")
					}
					time.Sleep(time.Millisecond)
				}
			}
			for _, a := range args {
				want := make([]byte, len(a.Pixels))
				for y := 0; y < a.Region.Dy(); y++ {
					for x := 0; x < a.Region.Dx(); x++ {
						xx, yy := x+a.Region.Min.X, y+a.Region.Min.Y
						copy(want[4*(y*a.Region.Dx()+x):], []byte{byte(xx), byte(yy), byte(xx + yy), 255})
					}
				}
				if !bytes.Equal(a.Pixels, want) {
					t.Errorf("pixels differ for region %v", a.Region)
				}
			}
			if !bytes.Equal(secondArgs[0].Pixels, make([]byte, len(secondArgs[0].Pixels))) {
				t.Error("subsequent write was not captured")
			}
			if err := g.End(graphicsdriver.FlushModeEndFrame); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestReadPixelsAsyncAbort(t *testing.T) {
	for _, version := range []string{"version=11", "version=12"} {
		t.Run(version, func(t *testing.T) {
			t.Setenv("EBITENGINE_DIRECTX", os.Getenv("EBITENGINE_DIRECTX")+","+version)
			g, err := directx.NewGraphics()
			if err != nil {
				t.Fatal(err)
			}
			if err := g.Begin(); err != nil {
				t.Fatal(err)
			}
			img, err := g.NewImage(257, 131)
			if err != nil {
				t.Fatal(err)
			}
			reader, ok := img.(graphicsdriver.AsyncPixelsReader)
			if !ok {
				t.Fatal("DirectX does not implement AsyncPixelsReader")
			}
			if _, err := reader.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{Region: image.Rect(0, 0, 3, 5), Pixels: make([]byte, 59)}}); err == nil {
				t.Error("invalid destination size was accepted")
			}
			for j := 0; j < 32; j++ {
				r, err := reader.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{Region: image.Rect(1, 3, 256, 130), Pixels: make([]byte, 4*255*127)}})
				if err != nil {
					t.Fatal(err)
				}
				r.Dispose()
			}
			img.Dispose()
			if err := g.End(graphicsdriver.FlushModeEndFrame); err != nil {
				t.Fatal(err)
			}
			if err := g.Begin(); err != nil {
				t.Fatal(err)
			}
			if err := g.End(graphicsdriver.FlushModeEndFrame); err != nil {
				t.Fatal(err)
			}
		})
	}
}
