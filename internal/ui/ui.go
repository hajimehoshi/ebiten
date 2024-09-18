// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"errors"
	"image"
	"sync"
	"sync/atomic"

	_ "github.com/ebitengine/hideconsole"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

// RegularTermination represents a regular termination.
// Run can return this error, and if this error is received,
// the game loop should be terminated as soon as possible.
var RegularTermination = errors.New("regular termination")

type FPSModeType int

const (
	FPSModeVsyncOn FPSModeType = iota
	FPSModeVsyncOffMaximum
	FPSModeVsyncOffMinimum
)

type CursorMode int

const (
	CursorModeVisible CursorMode = iota
	CursorModeHidden
	CursorModeCaptured
)

type CursorShape int

const (
	CursorShapeDefault CursorShape = iota
	CursorShapeText
	CursorShapeCrosshair
	CursorShapePointer
	CursorShapeEWResize
	CursorShapeNSResize
	CursorShapeNESWResize
	CursorShapeNWSEResize
	CursorShapeMove
	CursorShapeNotAllowed
)

type WindowResizingMode int

const (
	WindowResizingModeDisabled WindowResizingMode = iota
	WindowResizingModeOnlyFullscreenEnabled
	WindowResizingModeEnabled
)

type UserInterface struct {
	err  error
	errM sync.Mutex

	isScreenClearedEveryFrame atomic.Bool
	graphicsLibrary           atomic.Int32
	running                   atomic.Bool
	terminated                atomic.Bool
	tick                      atomic.Uint64

	whiteImage *Image

	mainThread thread.Thread

	userInterfaceImpl
}

var (
	theUI *UserInterface
)

func init() {
	// newUserInterface() must be called in the main goroutine.
	u, err := newUserInterface()
	if err != nil {
		panic(err)
	}
	theUI = u
}

func Get() *UserInterface {
	return theUI
}

// newUserInterface must be called from the main thread.
func newUserInterface() (*UserInterface, error) {
	u := &UserInterface{}
	u.isScreenClearedEveryFrame.Store(true)
	u.graphicsLibrary.Store(int32(GraphicsLibraryUnknown))

	u.whiteImage = u.NewImage(3, 3, atlas.ImageTypeRegular)
	pix := make([]byte, 4*u.whiteImage.width*u.whiteImage.height)
	for i := range pix {
		pix[i] = 0xff
	}
	// As a white image is used at Fill, use WritePixels instead.
	u.whiteImage.WritePixels(pix, image.Rect(0, 0, u.whiteImage.width, u.whiteImage.height))

	if err := u.init(); err != nil {
		return nil, err
	}

	return u, nil
}

func (u *UserInterface) readPixels(mipmap *mipmap.Mipmap, pixels []byte, region image.Rectangle) error {
	if !u.running.Load() {
		panic("ui: ReadPixels cannot be called before the game starts")
	}

	ok, err := mipmap.ReadPixels(u.graphicsDriver, pixels, region)
	if err != nil {
		return err
	}

	// ReadPixels failed since this was called in between two frames.
	// Try this again at the next frame.
	if !ok {
		// If this function is called from the same sequence as a game's Update and Draw,
		// this causes a dead lock.
		// This never happens so far, but if handling inputs after EndFrame is implemented,
		// this might be possible (#1704).

		var err1 error
		u.context.runInFrame(func() {
			ok, err := mipmap.ReadPixels(u.graphicsDriver, pixels, region)
			if err != nil {
				err1 = err
				return
			}
			if !ok {
				// This never reaches since this function must be called in a frame.
				panic("ui: ReadPixels unexpectedly failed")
			}
		})
		return err1
	}

	return nil
}

func (u *UserInterface) dumpScreenshot(mipmap *mipmap.Mipmap, name string, blackbg bool) (string, error) {
	return mipmap.DumpScreenshot(u.graphicsDriver, name, blackbg)
}

func (u *UserInterface) dumpImages(dir string) (string, error) {
	return atlas.DumpImages(u.graphicsDriver, dir)
}

type RunOptions struct {
	GraphicsLibrary          GraphicsLibrary
	InitUnfocused            bool
	ScreenTransparent        bool
	SkipTaskbar              bool
	SingleThread             bool
	DisableHiDPI             bool
	ColorSpace               graphicsdriver.ColorSpace
	X11ClassName             string
	X11InstanceName          string
	StrictContextRestoration bool
}

// InitialWindowPosition returns the position for centering the given second width/height pair within the first width/height pair.
func InitialWindowPosition(mw, mh, ww, wh int) (x, y int) {
	return (mw - ww) / 2, (mh - wh) / 3
}

func (u *UserInterface) error() error {
	u.errM.Lock()
	defer u.errM.Unlock()
	return u.err
}

func (u *UserInterface) setError(err error) {
	u.errM.Lock()
	defer u.errM.Unlock()
	if u.err == nil {
		u.err = err
	}
}

func (u *UserInterface) IsScreenClearedEveryFrame() bool {
	return u.isScreenClearedEveryFrame.Load()
}

func (u *UserInterface) SetScreenClearedEveryFrame(cleared bool) {
	u.isScreenClearedEveryFrame.Store(cleared)
}

func (u *UserInterface) setGraphicsLibrary(library GraphicsLibrary) {
	u.graphicsLibrary.Store(int32(library))
}

func (u *UserInterface) GraphicsLibrary() GraphicsLibrary {
	return GraphicsLibrary(u.graphicsLibrary.Load())
}

func (u *UserInterface) isRunning() bool {
	return u.running.Load() && !u.isTerminated()
}

func (u *UserInterface) setRunning(running bool) {
	u.running.Store(running)
}

func (u *UserInterface) isTerminated() bool {
	return u.terminated.Load()
}

func (u *UserInterface) setTerminated() {
	u.terminated.Store(true)
}

func (u *UserInterface) Tick() uint64 {
	return u.tick.Load()
}
