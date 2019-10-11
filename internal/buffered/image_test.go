// Copyright 2019 The Ebiten Authors
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

package buffered_test

import (
	"errors"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten"
)

var mainCh = make(chan func())

func runOnMainThread(f func()) {
	ch := make(chan struct{})
	mainCh <- func() {
		f()
		close(ch)
	}
	<-ch
}

func TestMain(m *testing.M) {
	go func() {
		os.Exit(m.Run())
	}()

	for {
		select {
		case f := <-mainCh:
			f()
		}
	}
}

func TestSetBeforeRun(t *testing.T) {
	clr := color.RGBA{1, 2, 3, 4}

	img, _ := ebiten.NewImage(16, 16, ebiten.FilterDefault)
	img.Set(0, 0, clr)

	want := clr
	var got color.RGBA

	runOnMainThread(func() {
		quit := errors.New("quit")
		if err := ebiten.Run(func(*ebiten.Image) error {
			got = img.At(0, 0).(color.RGBA)
			return quit
		}, 320, 240, 1, ""); err != nil && err != quit {
			t.Fatal(err)
		}
	})

	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
