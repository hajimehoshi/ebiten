// Copyright 2016 Hajime Hoshi
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

package ebiten_test

import (
	"image"
	_ "image/png"
	"testing"

	. "github.com/hajimehoshi/ebiten"
)

var ebitenImageBin = ""

func openImage(path string) (image.Image, error) {
	file, err := readFile(path)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func openEbitenImage(path string) (*Image, image.Image, error) {
	img, err := openImage(path)
	if err != nil {
		return nil, nil, err
	}

	eimg, err := NewImageFromImage(img, FilterNearest)
	if err != nil {
		return nil, nil, err
	}
	return eimg, img, nil
}

func TestImageSelf(t *testing.T) {
	img, _, err := openEbitenImage("testdata/ebiten.png")
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := img.DrawImage(img, nil); err == nil {
		t.Fatalf("img.DrawImage(img, nil) doesn't return error; an error should be returned")
	}
}

func TestImageDispose(t *testing.T) {
	img, err := NewImage(16, 16, FilterNearest)
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := img.Dispose(); err != nil {
		t.Errorf("img.Dipose() returns error: %v", err)
	}
}

func TestNewImageFromEbitenImage(t *testing.T) {
	img, _, err := openEbitenImage("testdata/ebiten.png")
	if err != nil {
		t.Fatal(err)
		return
	}
	if _, err := NewImageFromImage(img, FilterNearest); err == nil {
		t.Errorf("NewImageFromImage with an *ebiten.Image must return an error")
	}
}
