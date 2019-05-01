// Copyright 2018 The Ebiten Authors
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

// +build js
// +build example jsgo

package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"math"
	"syscall/js"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	count        = 0
	gophersImage *ebiten.Image
	extraImages  []*ebiten.Image
	lost         = false
)

func loseAndRestoreContext(context js.Value) {
	if lost {
		return
	}

	// Edge might not support the extension. See
	// https://developer.mozilla.org/en-US/docs/Web/API/WEBGL_lose_context
	ext := context.Call("getExtension", "WEBGL_lose_context")
	if ext == js.Null() {
		fmt.Println("Fail to force context lost. Edge might not support the extension yet.")
		return
	}

	ext.Call("loseContext")
	fmt.Println("Lost the context!")
	fmt.Println("The context is automatically restored after 3 seconds.")
	lost = true

	// If and only if the context is lost by loseContext, you need to call restoreContext. Note that in usual
	// case of context lost, you cannot call restoreContext but the context should be restored automatically.
	//
	// After the context is lost, update will not be called. Instead, fire the goroutine to restore the context.
	go func() {
		time.Sleep(3 * time.Second)
		ext.Call("restoreContext")
		fmt.Println("Restored the context!")
		lost = false
	}()
}

func update(screen *ebiten.Image) error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		doc := js.Global().Get("document")
		canvas := doc.Call("getElementsByTagName", "canvas").Index(0)
		context := canvas.Call("getContext", "webgl")
		if context == js.Null() {
			context = canvas.Call("getContext", "experimental-webgl")
		}

		loseAndRestoreContext(context)
		return nil
	}

	count++
	if ebiten.IsDrawingSkipped() {
		return nil
	}
	w, h := gophersImage.Size()
	op := &ebiten.DrawImageOptions{}

	// For the details, see examples/rotate.
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Rotate(float64(count%360) * 2 * math.Pi / 360)
	op.GeoM.Translate(screenWidth/2, screenHeight/2)
	screen.DrawImage(gophersImage, op)

	ebitenutil.DebugPrint(screen, "Press Space to force to lose/restore the GL context!\n(Browser only)")

	return nil
}

func main() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	// Extend the shared backend GL texture on purpose.
	for i := 0; i < 20; i++ {
		eimg, _ := ebiten.NewImageFromImage(img, ebiten.FilterDefault)
		extraImages = append(extraImages, eimg)
	}

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Context Lost (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
