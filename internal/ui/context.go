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
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

var (
	NearestFilterShader = &Shader{shader: atlas.NearestFilterShader}
	LinearFilterShader  = &Shader{shader: atlas.LinearFilterShader}
)

type Game interface {
	NewOffscreenImage(width, height int) *Image
	NewScreenImage(width, height int) *Image
	Layout(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64)
	UpdateInputState(fn func(*InputState))
	Update() error
	DrawOffscreen() error
	DrawFinalScreen(scale, offsetX, offsetY float64)
}

type context struct {
	game Game

	screenTransparent bool

	updateCalled bool

	offscreen *Image
	screen    *Image

	// screenWidth and screenHeight are the size of the final rendering destination, in pixels.
	// They are given by the platform rather than derived from the outside size, which cannot
	// represent the pixel count exactly at a fractional device scale factor.
	screenWidth  int
	screenHeight int

	offscreenWidth  float64
	offscreenHeight float64

	isOffscreenModified bool

	// virtualKeyboardOffsetY shifts the screen so that a virtual keyboard does not cover the
	// text-input caret. It is refreshed once per frame, so that everything derived from the
	// screen transform within a frame agrees.
	virtualKeyboardOffsetY float64

	// offscreenDrawn reports whether the current offscreen has been drawn since it was
	// created. The offscreen keeps its content across frames, but is recreated empty when
	// the screen size changes, so this guards against presenting an empty (black) frame.
	offscreenDrawn bool

	lastSwapBufferTime time.Time

	// vsyncIgnored reports whether swapping buffers does not wait for the display even though vsync
	// is enabled. The loop must then be paced explicitly.
	vsyncIgnored bool

	// vsyncIgnoredCount is the number of the successive frames that were swapped too early for the
	// display to have shown them.
	vsyncIgnoredCount int

	skipCount int

	funcsInFrameCh chan func()
}

func newContext(game Game, screenTransparent bool) *context {
	return &context{
		game:              game,
		screenTransparent: screenTransparent,
		funcsInFrameCh:    make(chan func()),
	}
}

// updateFrame runs one frame. present reports whether the frame should be shown on the screen; when it
// is false (e.g. the window is hidden) the buffer swap is skipped, so the loop paces from
// swapBuffersOrWait's no-swap path instead of a present that may block, and Update keeps running at the
// target tick rate.
func (c *context) updateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, screenWidth, screenHeight int, deviceScaleFactor float64, ui *UserInterface, present bool) error {
	// TODO: If updateCount is 0 and vsync is disabled, swapping buffers can be skipped.
	needsSwapBuffers, err := c.updateFrameImpl(graphicsDriver, clock.UpdateFrame(), outsideWidth, outsideHeight, screenWidth, screenHeight, deviceScaleFactor, ui, false)
	if err != nil {
		return err
	}
	if err := c.swapBuffersOrWait(needsSwapBuffers && present, graphicsDriver, ui.FPSMode() == FPSModeVsyncOn, ui.RefreshRate()); err != nil {
		return err
	}
	return nil
}

// forceUpdateFrame runs one frame with a forced draw, regardless of the draw-skipping states.
// The number of ticks is determined by the clock like updateFrame.
func (c *context) forceUpdateFrame(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight float64, screenWidth, screenHeight int, deviceScaleFactor float64, ui *UserInterface) error {
	n := 1
	if ui.GraphicsLibrary() == GraphicsLibraryDirectX {
		// On DirectX, both framebuffers in the swap chain should be updated.
		// Or, the rendering result becomes unexpected when the window is resized.
		n = 2
	}
	for range n {
		// Let the clock determine the tick count as usual, instead of forcing one tick per call.
		// Forced frames can happen at any rate, like once per window-resizing event, and forcing
		// a tick there would advance the game time faster than the specified TPS (#2615).
		needsSwapBuffers, err := c.updateFrameImpl(graphicsDriver, clock.UpdateFrame(), outsideWidth, outsideHeight, screenWidth, screenHeight, deviceScaleFactor, ui, true)
		if err != nil {
			return err
		}
		if err := c.swapBuffersOrWait(needsSwapBuffers, graphicsDriver, ui.FPSMode() == FPSModeVsyncOn, ui.RefreshRate()); err != nil {
			return err
		}
	}

	// A pipelined driver (DirectX 12) may not finish a forced frame before control returns to the
	// OS's window-resize loop, showing stale content. Wait for it to finish if supported (#3477).
	if err := graphicscommand.FinishForcedFrame(graphicsDriver); err != nil {
		return err
	}
	return nil
}

func (c *context) updateFrameImpl(graphicsDriver graphicsdriver.Graphics, updateCount int, outsideWidth, outsideHeight float64, screenWidth, screenHeight int, deviceScaleFactor float64, ui *UserInterface, forceDraw bool) (needsSwapBuffers bool, err error) {
	// The given outside size can be 0 e.g. just after restoring from the fullscreen mode on Windows (#1589)
	// Just ignore such cases. Otherwise, creating a zero-sized framebuffer causes a panic.
	if outsideWidth == 0 || outsideHeight == 0 {
		return false, nil
	}

	// On Android, GPU resources might be saved until the app is resumed.
	// Skip updating and drawing until then.
	if atlas.IsSuspended() {
		return true, nil
	}

	debug.FrameLogf("----\n")

	if err := atlas.BeginFrame(graphicsDriver); err != nil {
		return false, err
	}
	defer func() {
		if atlasErr := atlas.EndFrame(graphicsDriver); atlasErr != nil {
			needsSwapBuffers = false
			err = errors.Join(err, atlasErr)
			return
		}
	}()

	// Flush deferred functions, like reading pixels from GPU.
	if err := c.processFuncsInFrame(ui); err != nil {
		return false, err
	}

	// ForceUpdate can be invoked even if the context is not initialized yet (#1591).
	ow, oh, resized := c.layoutGame(outsideWidth, outsideHeight, screenWidth, screenHeight)
	if ow == 0 || oh == 0 {
		return false, nil
	}

	// Refresh the shift after the layout, which it is derived from, and before the input state,
	// which is converted with it.
	c.updateVirtualKeyboardOffsetY()

	// Update the input state after the layout is updated as a cursor position is affected by the layout.
	if err := ui.updateInputStateForFrame(deviceScaleFactor); err != nil {
		return false, err
	}

	// Ensure Update runs at least once before the first Draw (so Update can be used for
	// initialization), and at least once whenever the layout size changes so that a resize
	// frame is always preceded by an Update at the new size. Without the latter, an incremental
	// renderer that only paints the regions computed during Update leaves the newly exposed area
	// unpainted while the window is being resized (#3477).
	if updateCount == 0 && (!c.updateCalled || resized) {
		updateCount = 1
		c.updateCalled = true
		// Reconcile the forced tick with the clock: forcing an Update on every resize step
		// (#3477) would otherwise advance the game time faster than the target TPS (#2615).
		clock.SinkTick()
	}
	debug.FrameLogf("Update count per frame: %d\n", updateCount)

	// Update the game.
	for range updateCount {
		c.readInputStateForTick(ui)

		if err := hook.RunBeforeUpdateHooks(); err != nil {
			return false, err
		}
		// This is not a virtualization guest, so the pre-tick hooks get false; the guest backend runs
		// them with true in updateTickForVMGuest.
		if err := hook.RunBeforeUpdateHooksWithVMGuestInfo(false); err != nil {
			return false, err
		}
		if err := c.game.Update(); err != nil {
			return false, err
		}

		// Catch the error that happened at (*Image).At.
		if err := ui.error(); err != nil {
			return false, err
		}

		ui.incrementTick()
	}

	// Update window icons during a frame, since an icon might be *ebiten.Image and
	// getting pixels from it needs to be in a frame (#1468).
	if err := ui.updateIconIfNeeded(); err != nil {
		return false, err
	}

	// Draw the game.
	return c.drawGame(graphicsDriver, ui, forceDraw)
}

// readInputStateForTick takes the input snapshot for the tick that is about to run.
func (c *context) readInputStateForTick(ui *UserInterface) {
	// Read the input state and use it for one tick to give a consistent result for one tick (#2496, #2501).
	c.game.UpdateInputState(func(inputState *InputState) {
		ui.readInputState(inputState)
	})
	// The snapshot for this tick is taken, so an event recorded from now on belongs to the next tick.
	ui.advanceInputTimeToNextTick()
}

func (c *context) swapBuffersOrWait(needsSwapBuffers bool, graphicsDriver graphicsdriver.Graphics, vsyncEnabled bool, refreshRate int) error {
	if needsSwapBuffers {
		if err := atlas.SwapBuffers(graphicsDriver); err != nil {
			return err
		}
	}

	// Swapping buffers for an invisible screen returns without waiting for the display. Pace such a
	// frame like a skipped swap, or the loop would run as fast as the CPU allows (#2181).
	var occluded bool
	if o, ok := graphicsDriver.(interface{ IsOccluded() bool }); ok {
		occluded = o.IsOccluded()
	}

	now := time.Now()

	// A frame that is not swapped with vsync enabled tells nothing about whether swapping buffers
	// waits for the display, and breaks the run of the frames that returned early.
	if !needsSwapBuffers || occluded || !vsyncEnabled {
		c.vsyncIgnoredCount = 0
	}

	var waitTime time.Duration
	if !needsSwapBuffers || occluded {
		// When swapping buffers is skipped and Draw is called too early, sleep for a while to suppress CPU usages (#2890).
		waitTime = time.Second / 60
	} else if vsyncEnabled {
		// In some environments, e.g. Linux on Parallels, SwapBuffers doesn't wait for the vsync (#2952).
		// In the case when the display has high refresh rates like 240 [Hz], the wait time should be small.
		waitTime = time.Millisecond

		// A graphics driver can be configured to force the vsync off, and then swapping buffers never
		// waits for the display (#3009). Pace the loop at the refresh rate, as nothing else does.
		if refreshRate > 0 {
			refreshInterval := time.Second / time.Duration(refreshRate)
			if c.updateVsyncIgnored(now.Sub(c.lastSwapBufferTime), refreshInterval) {
				waitTime = refreshInterval
			}
		}
	}

	// Pace with an absolute deadline to avoid drift.
	if waitTime > 0 {
		if next := c.lastSwapBufferTime.Add(waitTime); next.After(now) {
			time.Sleep(next.Sub(now))
			c.lastSwapBufferTime = next
			return nil
		}
	}
	c.lastSwapBufferTime = now

	return nil
}

// resetVsyncDetection makes whether swapping buffers waits for the display measured again.
func (c *context) resetVsyncDetection() {
	c.vsyncIgnored = false
	c.vsyncIgnoredCount = 0
}

// updateVsyncIgnored records how long one frame took, including swapping its buffers, and reports
// whether swapping buffers turns out not to wait for the display.
func (c *context) updateVsyncIgnored(frameTime, refreshInterval time.Duration) bool {
	if c.vsyncIgnored {
		return true
	}

	// A swap can return early while the graphics driver still has room to queue frames. Require a
	// long run of early frames to tell an environment that never waits from such a burst.
	const threshold = 30

	if frameTime >= refreshInterval/2 {
		c.vsyncIgnoredCount = 0
		return false
	}

	c.vsyncIgnoredCount++
	if c.vsyncIgnoredCount < threshold {
		return false
	}

	c.vsyncIgnored = true
	return true
}

func (c *context) newOffscreenImage(w, h int) *Image {
	img := c.game.NewOffscreenImage(w, h)
	img.modifyCallback = func() {
		c.isOffscreenModified = true
	}
	// A newly created offscreen has no content yet.
	c.offscreenDrawn = false
	return img
}

func (c *context) drawGame(graphicsDriver graphicsdriver.Graphics, ui *UserInterface, forceDraw bool) (needSwapBuffers bool, err error) {
	// isOffscreenModified is updated when an offscreen's modifyCallback.
	c.isOffscreenModified = false

	// Even though updateCount == 0, the offscreen is cleared and Draw is called.
	// Draw should not update the game state and then the screen should not be updated without Update, but
	// users might want to process something at Draw with the time intervals of FPS.
	if ui.IsScreenClearedEveryFrame() {
		c.offscreen.clear()
	}

	if err := c.game.DrawOffscreen(); err != nil {
		return false, err
	}

	if c.isOffscreenModified {
		c.offscreenDrawn = true
	}

	const maxSkipCount = 4

	if !forceDraw && !c.isOffscreenModified {
		if c.skipCount < maxSkipCount {
			c.skipCount++
		}
	} else {
		c.skipCount = 0
	}

	if c.skipCount >= maxSkipCount {
		return false, nil
	}

	// Never present an offscreen that has not been drawn since it was recreated on a resize.
	// A game that repaints only on change (an incremental renderer) may draw nothing on the
	// frame the offscreen is recreated, and presenting it would flash an empty, black frame.
	// Keep the previously presented frame until the game repaints (#3477). Like the skip-count
	// above, this is bypassed by a forced draw.
	if !forceDraw && !c.offscreenDrawn {
		return false, nil
	}

	// screen can be nil for some edge cases (#3121).
	if c.screen == nil {
		return false, nil
	}

	if graphicsDriver.NeedsClearingScreen() {
		// This clear is needed for fullscreen mode or some mobile platforms (#622).
		// An opaque screen is cleared with opaque black: when the screen's framebuffer has an alpha
		// channel, a compositor would show the desktop through the area the offscreen does not
		// cover (#3454).
		if c.screenTransparent {
			c.screen.clear()
		} else {
			c.screen.fillBlack()
		}
	}

	c.game.DrawFinalScreen(c.screenScaleAndOffsets())

	return true, nil
}

// layoutGame updates the game's layout for the given outside size, in device-independent pixels,
// and the given screen size, in pixels. It returns the offscreen size in pixels. The returned bool
// reports whether the screen size changed from the previous layout.
func (c *context) layoutGame(outsideWidth, outsideHeight float64, screenWidth, screenHeight int) (int, int, bool) {
	owf, ohf := c.game.Layout(outsideWidth, outsideHeight)
	if owf <= 0 || ohf <= 0 {
		panic("ui: Layout must return positive numbers")
	}

	resized := c.screenWidth != screenWidth || c.screenHeight != screenHeight
	if resized {
		c.skipCount = 0
	}
	c.screenWidth = screenWidth
	c.screenHeight = screenHeight
	c.offscreenWidth = owf
	c.offscreenHeight = ohf

	ow := int(math.Ceil(c.offscreenWidth))
	oh := int(math.Ceil(c.offscreenHeight))

	if c.screen != nil && (c.screen.width != screenWidth || c.screen.height != screenHeight) {
		c.screen.Deallocate()
		c.screen = nil
	}
	if c.screen == nil && screenWidth > 0 && screenHeight > 0 {
		c.screen = c.game.NewScreenImage(screenWidth, screenHeight)
	}

	if c.offscreen != nil && (c.offscreen.width != ow || c.offscreen.height != oh) {
		c.offscreen.Deallocate()
		c.offscreen = nil
	}
	if c.offscreen == nil {
		c.offscreen = c.newOffscreenImage(ow, oh)
	}

	return ow, oh, resized
}

func (c *context) clientPositionToLogicalPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	s, ox, oy := c.screenScaleAndOffsets()
	// The scale 0 indicates that the screen is not initialized yet.
	// As any cursor values don't make sense, just return NaN.
	if s == 0 {
		return math.NaN(), math.NaN()
	}
	return (x*deviceScaleFactor - ox) / s, (y*deviceScaleFactor - oy) / s
}

func (c *context) logicalPositionToClientPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	s, ox, oy := c.screenScaleAndOffsets()
	return (x*s + ox) / deviceScaleFactor, (y*s + oy) / deviceScaleFactor
}

// screenScaleAndOffsets returns the transform from the offscreen to the screen. It includes the
// shift avoiding a virtual keyboard, so that the rendering, the input positions and the caret
// position reported to the platform IME all agree.
func (c *context) screenScaleAndOffsets() (scale, offsetX, offsetY float64) {
	scale, offsetX, offsetY = c.letterboxScaleAndOffsets()
	return scale, offsetX, offsetY + c.virtualKeyboardOffsetY
}

// letterboxScaleAndOffsets returns the transform centering the offscreen in the screen.
func (c *context) letterboxScaleAndOffsets() (scale, offsetX, offsetY float64) {
	sw := float64(c.screenWidth)
	sh := float64(c.screenHeight)
	scaleX := sw / c.offscreenWidth
	scaleY := sh / c.offscreenHeight
	scale = min(scaleX, scaleY)
	width := c.offscreenWidth * scale
	height := c.offscreenHeight * scale
	offsetX = (sw - width) / 2
	offsetY = (sh - height) / 2
	return
}

// updateVirtualKeyboardOffsetY refreshes the shift lifting the text-input caret out of the
// region a virtual keyboard covers. The shift is zero or negative, and for a caret on the
// screen it never exceeds the covered region's height, so the strip it exposes at the bottom
// of the screen stays under the keyboard.
//
// The shift is derived from the letterbox transform rather than the current one: a shift
// measured against an already shifted screen would feed back into itself.
func (c *context) updateVirtualKeyboardOffsetY() {
	c.virtualKeyboardOffsetY = 0

	caretBounds, visibleRegion, ok := theVirtualKeyboard.state()
	if !ok {
		return
	}

	scale, _, offsetY := c.letterboxScaleAndOffsets()
	// The scale 0 indicates that the screen is not initialized yet.
	if scale == 0 {
		return
	}
	caretBottom := float64(caretBounds.Max.Y)*scale + offsetY
	visibleBottom := float64(visibleRegion.Max.Y)
	if caretBottom <= visibleBottom {
		return
	}
	c.virtualKeyboardOffsetY = visibleBottom - caretBottom
}

// monitorDeviceScaleFactor returns the current monitor's device scale factor, or 1 when no monitor
// is available.
func (u *UserInterface) monitorDeviceScaleFactor() float64 {
	m := u.Monitor()
	if m == nil {
		return 1
	}
	return m.DeviceScaleFactor()
}

func (u *UserInterface) LogicalPositionToClientPositionInNativePixels(x, y float64) (float64, float64) {
	s := u.monitorDeviceScaleFactor()
	x, y = u.context.logicalPositionToClientPosition(x, y, s)
	x = dipToNativePixels(x, s)
	y = dipToNativePixels(y, s)
	return x, y
}

// LogicalPositionToClientPositionInDIPs converts a logical position to a client-area position in
// device-independent pixels, which mean the same lengths on every platform (unlike native pixels).
func (u *UserInterface) LogicalPositionToClientPositionInDIPs(x, y float64) (float64, float64) {
	return u.context.logicalPositionToClientPosition(x, y, u.monitorDeviceScaleFactor())
}

func (c *context) runInFrame(f func()) {
	ch := make(chan struct{})
	c.funcsInFrameCh <- func() {
		defer close(ch)
		f()
	}
	<-ch
}

func (c *context) processFuncsInFrame(ui *UserInterface) error {
	var processed bool
	for {
		select {
		case f := <-c.funcsInFrameCh:
			f()
			processed = true
		default:
			if processed {
				// Catch the error that happened at (*Image).At.
				if err := ui.error(); err != nil {
					return err
				}
			}
			return nil
		}
	}
}
