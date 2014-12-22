/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten_test

import (
	. "."
	"image"
	"image/color"
	_ "image/png"
	"os"
	"testing"
)

func TestNewImageFromImage(t *testing.T) {
	file, err := os.Open("testdata/ebiten.png")
	if err != nil {
		t.Fatal(err)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatal(err)
		return
	}

	eimg, err := NewImageFromImage(img, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}

	if got := eimg.Bounds().Size(); got != img.Bounds().Size() {
		t.Errorf("img size: got %d; want %d", got, img.Bounds().Size())
	}

	for j := 0; j < eimg.Bounds().Size().Y; j++ {
		for i := 0; i < eimg.Bounds().Size().X; i++ {
			got := eimg.At(i, j)
			want := color.RGBAModel.Convert(img.At(i, j))
			if got != want {
				t.Errorf("img(%d, %d): got %#v; want %#v", i, j, got, want)
			}
		}
	}
}

// TODO: Add more tests (e.g. DrawImage with color matrix)
