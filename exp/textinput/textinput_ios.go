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

package textinput

import (
	"image"
	"sync"

	"github.com/ebitengine/purego/cstrings"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// The UIKit text input integration mirrors the browser backend: a hidden
// UITextView, seeded with the session's surrounding text, is made the first
// responder, and every change reported by its delegate is diffed against the
// session's baseline by diffSender.
//
// UIKit must be used on the main thread, while Start runs on the game's
// thread. Main-thread work is dispatched with performSelectorOnMainThread and
// reads its inputs from mutex-guarded fields. A generation counter guards the
// opposite direction: delegate events reported between a Start and the
// dispatched seeding describe the previous target and are dropped. The counter
// also dates a dismissal, which a Start between the request and the dispatched
// resigning cancels: resigning would end the session that Start just opened.
//
// Cancelling a session abandons its seed: a seeding dispatched before the
// cancellation neither opens the virtual keyboard nor lets the abandoned
// target's events through.
//
// Some keys are not text: the text view is a subclass whose insertText: and
// deleteBackward report the key press to the game instead of editing the text.

type cgPoint struct {
	x float64
	y float64
}

type cgSize struct {
	width  float64
	height float64
}

type cgRect struct {
	origin cgPoint
	size   cgSize
}

type nsRange struct {
	location uint
	length   uint
}

var (
	sel_CGRectValue                 = objc.RegisterName("CGRectValue")
	sel_addObject                   = objc.RegisterName("addObject:")
	sel_addObserver                 = objc.RegisterName("addObserver:selector:name:object:")
	sel_addSubview                  = objc.RegisterName("addSubview:")
	sel_alloc                       = objc.RegisterName("alloc")
	sel_allObjects                  = objc.RegisterName("allObjects")
	sel_count                       = objc.RegisterName("count")
	sel_applyDismiss                = objc.RegisterName("ebitengineApplyDismiss")
	sel_applyStart                  = objc.RegisterName("ebitengineApplyStart")
	sel_becomeFirstResponder        = objc.RegisterName("becomeFirstResponder")
	sel_bounds                      = objc.RegisterName("bounds")
	sel_convertRectFromView         = objc.RegisterName("convertRect:fromView:")
	sel_convertRectFromWindow       = objc.RegisterName("convertRect:fromWindow:")
	sel_defaultCenter               = objc.RegisterName("defaultCenter")
	sel_deleteBackward              = objc.RegisterName("deleteBackward")
	sel_init                        = objc.RegisterName("init")
	sel_initWithUTF8String          = objc.RegisterName("initWithUTF8String:")
	sel_insertText                  = objc.RegisterName("insertText:")
	sel_key                         = objc.RegisterName("key")
	sel_keyCode                     = objc.RegisterName("keyCode")
	sel_keyboardWillChangeFrame     = objc.RegisterName("ebitengineKeyboardWillChangeFrame:")
	sel_markedTextRange             = objc.RegisterName("markedTextRange")
	sel_objectAtIndex               = objc.RegisterName("objectAtIndex:")
	sel_objectForKey                = objc.RegisterName("objectForKey:")
	sel_performSelectorOnMainThread = objc.RegisterName("performSelectorOnMainThread:withObject:waitUntilDone:")
	sel_pressesBegan                = objc.RegisterName("pressesBegan:withEvent:")
	sel_pressesCancelled            = objc.RegisterName("pressesCancelled:withEvent:")
	sel_pressesEnded                = objc.RegisterName("pressesEnded:withEvent:")
	sel_release                     = objc.RegisterName("release")
	sel_resignFirstResponder        = objc.RegisterName("resignFirstResponder")
	sel_selectedRange               = objc.RegisterName("selectedRange")
	sel_setAutocapitalizationType   = objc.RegisterName("setAutocapitalizationType:")
	sel_setDelegate                 = objc.RegisterName("setDelegate:")
	sel_setFrame                    = objc.RegisterName("setFrame:")
	sel_setSelectedRange            = objc.RegisterName("setSelectedRange:")
	sel_setText                     = objc.RegisterName("setText:")
	sel_text                        = objc.RegisterName("text")
	sel_userInfo                    = objc.RegisterName("userInfo")
	sel_window                      = objc.RegisterName("window")
)

// newNSString returns a retained NSString. The caller releases it with
// sel_release unless it must stay alive.
func newNSString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(sel_alloc).Send(sel_initWithUTF8String, s+"\x00")
}

func init() {
	t := &theTextInputImpl
	t.sender.events = t.events
	hook.AppendHookOnBeforeUpdate(func() error {
		t.dismissVirtualKeyboardIfNeeded()
		return nil
	})
}

type textInputImpl struct {
	events *textInputEvents
	sender diffSender

	// pendingText, pendingCaretInUTF16, and pendingBounds hold the seed the
	// next ebitengineApplyStart applies on the main thread. generation is
	// bumped by Start and stamped into appliedGeneration when the seed is
	// applied; delegate events reported in between are dropped.
	pendingText         string
	pendingCaretInUTF16 int
	pendingBounds       image.Rectangle
	generation          int
	appliedGeneration   int

	// cancelled reports whether the current generation's target was abandoned.
	// The seed of a cancelled generation is not applied, and its delegate
	// events are dropped, until the next Start.
	cancelled bool

	// active reports whether text inputting has been requested and not
	// dismissed. closedTicks counts consecutive ticks without text inputting;
	// see dismissVirtualKeyboardIfNeeded. dismissGeneration is the generation the
	// dispatched dismissal was requested for.
	active            bool
	closedTicks       int
	dismissGeneration int

	// legacyCleared drops the delegate events fired by the legacy path
	// clearing the text view.
	legacyCleared bool

	// vkVisible and vkVisibleRegion are the virtual keyboard state reported by
	// UIKit, written on the main thread. vkKnown reports whether a keyboard
	// notification has been observed.
	vkVisible       bool
	vkVisibleRegion image.Rectangle
	vkKnown         bool

	// textView, delegate, parentView, and consumedPresses are used only on the
	// main thread, after ensureUIKit. consumedPresses maps an in-flight UIPress
	// consumed by pressesBeganOnMain to the key delivered to the game, so that
	// its ending releases the key and stays consumed too.
	textView        objc.ID
	delegate        objc.ID
	parentView      objc.ID
	consumedPresses map[objc.ID]ui.Key

	ensureUIKitOnce sync.Once

	mu sync.Mutex
}

var (
	class_EbitengineTextInputDelegate objc.Class
	class_EbitengineTextInputTextView objc.Class
)

// ensureUIKit registers the delegate class and the keyboard-frame observer on
// first use.
func (t *textInputImpl) ensureUIKit() {
	t.ensureUIKitOnce.Do(func() {
		var protocols []*objc.Protocol
		// The protocol is checked at runtime through respondsToSelector:, so a
		// missing protocol registration is not fatal.
		if p := objc.GetProtocol("UITextViewDelegate"); p != nil {
			protocols = append(protocols, p)
		}
		var err error
		class_EbitengineTextInputDelegate, err = objc.RegisterClass(
			"EbitengineTextInputDelegate",
			objc.GetClass("NSObject"),
			protocols,
			nil,
			[]objc.MethodDef{
				{
					Cmd: sel_applyStart,
					Fn: func(_ objc.ID, _ objc.SEL) {
						theTextInputImpl.applyStartOnMain()
					},
				},
				{
					Cmd: sel_applyDismiss,
					Fn: func(_ objc.ID, _ objc.SEL) {
						theTextInputImpl.applyDismissOnMain()
					},
				},
				{
					Cmd: objc.RegisterName("textViewDidChange:"),
					Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
						theTextInputImpl.textViewChangedOnMain()
					},
				},
				{
					Cmd: objc.RegisterName("textViewDidChangeSelection:"),
					Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
						theTextInputImpl.textViewChangedOnMain()
					},
				},
				{
					Cmd: objc.RegisterName("textViewDidEndEditing:"),
					Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
						theTextInputImpl.textViewEndedEditingOnMain()
					},
				},
				{
					Cmd: sel_keyboardWillChangeFrame,
					Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
						theTextInputImpl.keyboardWillChangeFrameOnMain(notification)
					},
				},
			},
		)
		if err != nil {
			panic(err)
		}

		class_EbitengineTextInputTextView, err = objc.RegisterClass(
			"EbitengineTextInputTextView",
			objc.GetClass("UITextView"),
			nil,
			nil,
			[]objc.MethodDef{
				{
					Cmd: sel_insertText,
					Fn: func(self objc.ID, _ objc.SEL, text objc.ID) {
						if insertTextOnMain(self, text) {
							return
						}
						self.SendSuper(sel_insertText, text)
					},
				},
				{
					Cmd: sel_deleteBackward,
					Fn: func(self objc.ID, _ objc.SEL) {
						if deleteBackwardOnMain(self) {
							return
						}
						self.SendSuper(sel_deleteBackward)
					},
				},
				{
					Cmd: sel_pressesBegan,
					Fn: func(self objc.ID, _ objc.SEL, presses, event objc.ID) {
						theTextInputImpl.pressesOnMain(self, presses, event, sel_pressesBegan)
					},
				},
				{
					Cmd: sel_pressesEnded,
					Fn: func(self objc.ID, _ objc.SEL, presses, event objc.ID) {
						theTextInputImpl.pressesOnMain(self, presses, event, sel_pressesEnded)
					},
				},
				{
					Cmd: sel_pressesCancelled,
					Fn: func(self objc.ID, _ objc.SEL, presses, event objc.ID) {
						theTextInputImpl.pressesOnMain(self, presses, event, sel_pressesCancelled)
					},
				},
			},
		)
		if err != nil {
			panic(err)
		}

		t.delegate = objc.ID(class_EbitengineTextInputDelegate).Send(sel_alloc).Send(sel_init)

		// The keyboard notification names are the constants' string values.
		// The name strings are kept alive for the notification center.
		center := objc.ID(objc.GetClass("NSNotificationCenter")).Send(sel_defaultCenter)
		center.Send(sel_addObserver, t.delegate, sel_keyboardWillChangeFrame, newNSString("UIKeyboardWillChangeFrameNotification"), objc.ID(0))
	})
}

// ensureTextViewOnMain creates the hidden text view lazily and attaches it to
// parent.
func (t *textInputImpl) ensureTextViewOnMain(parent objc.ID) objc.ID {
	if t.textView == 0 {
		tv := objc.ID(class_EbitengineTextInputTextView).Send(sel_alloc).Send(sel_init)
		// UITextAutocapitalizationTypeNone
		tv.Send(sel_setAutocapitalizationType, 0)
		tv.Send(sel_setDelegate, t.delegate)
		t.textView = tv
	}
	if t.parentView != parent {
		parent.Send(sel_addSubview, t.textView)
		t.parentView = parent
	}
	return t.textView
}

func (t *textInputImpl) markIMEDiscardNeeded() {
	t.mu.Lock()
	defer t.mu.Unlock()
	// The composition itself needs no recording: ebitengineApplyStart always
	// rewrites the text view's content, which discards a leftover composition,
	// and dismissing clears it. The abandonment itself must be recorded: the
	// seeding the last Start dispatched runs on the main thread only after the
	// tick ends.
	t.cancelled = true
}

func (t *textInputImpl) Start(bounds image.Rectangle, textBeforeCaret, textAfterCaret string) (<-chan textInputState, func()) {
	bounds = caretBoundsInClientNativePixels(bounds)
	value := textBeforeCaret + textAfterCaret
	caret := max(convertByteCountToUTF16Count(textBeforeCaret, len(textBeforeCaret)), 0)
	t.setPendingStart(value, caret, bounds)

	ch, end := t.events.start()

	t.ensureUIKit()
	t.delegate.Send(sel_performSelectorOnMainThread, sel_applyStart, objc.ID(0), false)

	return ch, end
}

// setPendingStart records the seed for the next ebitengineApplyStart and makes
// value the diff baseline.
func (t *textInputImpl) setPendingStart(value string, caretInUTF16 int, bounds image.Rectangle) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pendingText = value
	t.pendingCaretInUTF16 = caretInUTF16
	t.pendingBounds = bounds
	t.generation++
	t.cancelled = false
	t.active = true
	t.closedTicks = 0
	t.sender.reset(value)
}

// pendingStart returns the seed recorded by the last Start, and whether its
// target was abandoned since.
func (t *textInputImpl) pendingStart() (text string, caretInUTF16 int, bounds image.Rectangle, generation int, cancelled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pendingText, t.pendingCaretInUTF16, t.pendingBounds, t.generation, t.cancelled
}

// markApplied records that the seed of generation was applied to the text view,
// and reports whether its target was abandoned while it was being applied.
func (t *textInputImpl) markApplied(generation int) (abandoned bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.appliedGeneration = generation
	// A newer generation is not this seed's business: the Start that bumped the
	// counter has its own seeding dispatched.
	return t.cancelled && generation == t.generation
}

// applyStartOnMain seeds the text view with the pending surrounding text and
// makes it the first responder.
func (t *textInputImpl) applyStartOnMain() {
	text, caret, bounds, gen, cancelled := t.pendingStart()
	if cancelled {
		// Seeding now would open the virtual keyboard for a target that is
		// gone. Leaving the generation unapplied drops the events the text
		// view reports for it.
		return
	}

	// The parent is the game's view, which the mobile binding reports on
	// startup. Nothing can be seeded before that.
	parent := objc.ID(ui.Get().UIView())
	if parent == 0 {
		return
	}
	tv := t.ensureTextViewOnMain(parent)

	// The text view is kept outside the window, where the browser and Android
	// backends keep theirs at the caret. UIKit scrolls the enclosing scroll view
	// of an embedding application to lift the first responder above the
	// keyboard; a text view outside the window is never covered by the keyboard,
	// so that scrolling never happens and the application's scroll position
	// stays put. The keyboard is avoided by shifting the rendering instead. An
	// onscreen text view would also draw over the game, and a hidden one could
	// not become the first responder. The size mirrors the caret bounds.
	tv.Send(sel_setFrame, cgRect{
		origin: cgPoint{x: -10000, y: -10000},
		size:   cgSize{width: float64(max(bounds.Dx(), 1)), height: float64(max(bounds.Dy(), 1))},
	})

	ns := newNSString(text)
	tv.Send(sel_setText, ns)
	ns.Send(sel_release)
	tv.Send(sel_setSelectedRange, nsRange{location: uint(caret), length: 0})
	tv.Send(sel_becomeFirstResponder)

	if t.markApplied(gen) {
		// The target was abandoned between the check above and the text view
		// becoming the first responder, so the virtual keyboard is open for a
		// target that is gone.
		t.resignAndClearOnMain()
	}
}

// applyDismissOnMain dismisses the virtual keyboard, unless a Start has made the
// dismissal obsolete.
func (t *textInputImpl) applyDismissOnMain() {
	if !t.shouldApplyDismiss() {
		return
	}
	t.resignAndClearOnMain()
}

// shouldApplyDismiss reports whether the dispatched dismissal still applies.
func (t *textInputImpl) shouldApplyDismiss() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	// A Start dispatched after the dismissal opens the virtual keyboard again,
	// and resigning in between would end the session it opened.
	return t.dismissGeneration == t.generation
}

// resignAndClearOnMain resigns the first responder, dismissing the virtual
// keyboard, and drops the text view's content along with any composition.
func (t *textInputImpl) resignAndClearOnMain() {
	if t.textView == 0 {
		return
	}
	t.textView.Send(sel_resignFirstResponder)
	ns := newNSString("")
	t.textView.Send(sel_setText, ns)
	ns.Send(sel_release)
}

// shouldDismiss reports whether the caller stopped inputting text and the
// virtual keyboard is to be dismissed.
func (t *textInputImpl) shouldDismiss() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.active {
		t.closedTicks = 0
		return false
	}
	if t.events.isOpen() || t.events.getActiveSession() != nil {
		t.closedTicks = 0
		return false
	}
	t.closedTicks++
	// This runs before Update, where text inputting reopens. Finishing text
	// inputting in one tick and reopening it in the next leaves one tick
	// closed, so two mean the caller stopped.
	if t.closedTicks < 2 {
		return false
	}
	t.closedTicks = 0
	t.active = false
	t.dismissGeneration = t.generation
	return true
}

// dismissVirtualKeyboardIfNeeded resigns the first responder once the caller
// stops inputting text, dismissing the virtual keyboard.
func (t *textInputImpl) dismissVirtualKeyboardIfNeeded() {
	if !t.shouldDismiss() {
		return
	}
	t.ensureUIKit()
	t.delegate.Send(sel_performSelectorOnMainThread, sel_applyDismiss, objc.ID(0), false)
	// Resigning fires events carrying the text the text view still holds. With
	// no session to receive them, they are queued for the next one.
	t.events.clearQueue()
}

// textViewChangedOnMain reads the text view and reports the edit. The text
// view is mutated only after handleTextViewChange returns, as UIKit can call
// the delegate back synchronously.
func (t *textInputImpl) textViewChangedOnMain() {
	tv := t.textView
	if tv == 0 {
		return
	}

	value := cstrings.NSStringToString(tv.Send(sel_text))
	sel := objc.Send[nsRange](tv, sel_selectedRange)
	kind := commitRegular
	if tv.Send(sel_markedTextRange) != 0 {
		kind = commitNone
	}

	if t.handleTextViewChange(value, int(sel.location), int(sel.location+sel.length), kind) {
		ns := newNSString("")
		tv.Send(sel_setText, ns)
		ns.Send(sel_release)
	}
}

// handleTextViewChange reports the edit and returns whether the caller must
// clear the text view.
func (t *textInputImpl) handleTextViewChange(value string, selStart, selEnd int, kind commitKind) (clearTextView bool) {
	// Evaluated outside t.mu: the focus lock is taken with no other lock held.
	fieldFocused := withFocusedField(func(*Field) {})

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.appliedGeneration != t.generation {
		// The event predates the seeding requested by the latest Start and
		// describes the previous target.
		return false
	}

	if t.cancelled {
		// The session the text view was seeded for was cancelled, so the event
		// describes an abandoned target.
		return false
	}

	// The selection does not track the preedit on iOS; see
	// compositionSelectionInBytes.
	return handlePlatformState(t.events, &t.sender, &t.legacyCleared, value, selStart, selEnd, true, kind, fieldFocused)
}

// insertTextOnMain reports whether text is the Return key, which the game
// receives as a key press and acts on itself. An insertion while the IME is
// composing finalizes the composition and is left to UIKit.
func insertTextOnMain(tv objc.ID, text objc.ID) (handled bool) {
	if cstrings.NSStringToString(text) != "\n" {
		return false
	}
	if tv.Send(sel_markedTextRange) != 0 {
		return false
	}
	dispatchKeyPressUnlessDown(ui.KeyEnter)
	return true
}

// deleteBackwardOnMain reports whether the deletion is at the head of the text
// view, where it reaches text the session does not expose: the game receives a
// Backspace key press and edits across the boundary itself.
func deleteBackwardOnMain(tv objc.ID) (handled bool) {
	if tv.Send(sel_markedTextRange) != 0 {
		return false
	}
	sel := objc.Send[nsRange](tv, sel_selectedRange)
	if sel.location != 0 || sel.length != 0 {
		return false
	}
	dispatchKeyPressUnlessDown(ui.KeyBackspace)
	return true
}

// dispatchKeyPressUnlessDown delivers a synthetic key press for an edit the
// virtual keyboard produced without a key event. A key that is already down
// was delivered by a real key event, so a synthetic press would double it.
func dispatchKeyPressUnlessDown(key ui.Key) {
	if ui.Get().IsKeyDown(key) {
		return
	}
	ui.Get().DispatchKeyPress(key)
}

// HID keyboard usages of the keys the text view consumes; see consumableKey.
const (
	hidUsageKeyboardReturn          = 0x28
	hidUsageKeyboardDeleteBackspace = 0x2A
)

// pressesOnMain handles a hardware key press event of the text view. A press
// of a key the game acts on itself (see consumableKey) is consumed: the game
// receives the key event, and the text system never edits the text for it, so
// nothing is doubled. The remaining presses follow the responder chain, which
// delivers them to the game's view controller and then to the text system.
func (t *textInputImpl) pressesOnMain(self, presses, event objc.ID, cmd objc.SEL) {
	arr := presses.Send(sel_allObjects)
	n := int(objc.Send[uint](arr, sel_count))

	var kept objc.ID
	consumed := 0
	for i := 0; i < n; i++ {
		p := arr.Send(sel_objectAtIndex, uint(i))
		if cmd == sel_pressesBegan {
			if key, ok := t.consumableKey(self, p); ok {
				if t.consumedPresses == nil {
					t.consumedPresses = map[objc.ID]ui.Key{}
				}
				t.consumedPresses[p] = key
				ui.Get().DispatchKeyDown(key)
				consumed++
				continue
			}
		} else if key, ok := t.consumedPresses[p]; ok {
			delete(t.consumedPresses, p)
			ui.Get().DispatchKeyUp(key)
			consumed++
			continue
		}
		if kept == 0 {
			kept = objc.ID(objc.GetClass("NSMutableSet")).Send(sel_alloc).Send(sel_init)
		}
		kept.Send(sel_addObject, p)
	}

	if consumed == 0 {
		if kept != 0 {
			kept.Send(sel_release)
		}
		self.SendSuper(cmd, presses, event)
		return
	}
	if kept != 0 {
		self.SendSuper(cmd, kept, event)
		kept.Send(sel_release)
	}
}

// consumableKey returns the key the game receives for press, and whether the
// press is consumed by the text view. A Return outside a composition is the
// game's: a line break is the game's decision. A Backspace at the head of the
// text view reaches text the session does not expose, and the game edits
// across the boundary itself.
func (t *textInputImpl) consumableKey(tv objc.ID, press objc.ID) (ui.Key, bool) {
	k := press.Send(sel_key)
	if k == 0 {
		return 0, false
	}
	switch objc.Send[int](k, sel_keyCode) {
	case hidUsageKeyboardReturn:
		if tv.Send(sel_markedTextRange) != 0 {
			return 0, false
		}
		return ui.KeyEnter, true
	case hidUsageKeyboardDeleteBackspace:
		if tv.Send(sel_markedTextRange) != 0 {
			return 0, false
		}
		sel := objc.Send[nsRange](tv, sel_selectedRange)
		if sel.location != 0 || sel.length != 0 {
			return 0, false
		}
		return ui.KeyBackspace, true
	}
	return 0, false
}

// textViewEndedEditingOnMain ends text inputting when the first responder is
// resigned, e.g. by the user dismissing the virtual keyboard.
func (t *textInputImpl) textViewEndedEditingOnMain() {
	if !t.markEndedEditing() {
		return
	}
	t.events.end()
}

// markEndedEditing records that the text view stopped editing, and reports
// whether the ending applies to the target the latest Start requested.
func (t *textInputImpl) markEndedEditing() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.appliedGeneration != t.generation {
		// The text view was resigned for the previous target, as the seeding the
		// latest Start dispatched has not run yet. Ending now would close the
		// session that Start opened.
		return false
	}
	t.active = false
	t.closedTicks = 0
	return true
}

// keyboardWillChangeFrameOnMain records the virtual keyboard state from a
// keyboard-frame notification.
func (t *textInputImpl) keyboardWillChangeFrameOnMain(notification objc.ID) {
	parent := objc.ID(ui.Get().UIView())
	if parent == 0 {
		return
	}
	window := parent.Send(sel_window)
	if window == 0 {
		return
	}

	userInfo := notification.Send(sel_userInfo)
	if userInfo == 0 {
		return
	}
	key := newNSString("UIKeyboardFrameEndUserInfoKey")
	frameValue := userInfo.Send(sel_objectForKey, key)
	key.Send(sel_release)
	if frameValue == 0 {
		return
	}
	// The keyboard frame is in the screen's coordinates.
	frame := objc.Send[cgRect](frameValue, sel_CGRectValue)
	windowFrame := objc.Send[cgRect](window, sel_convertRectFromWindow, frame, objc.ID(0))
	viewFrame := objc.Send[cgRect](parent, sel_convertRectFromView, windowFrame, window)
	bounds := objc.Send[cgRect](parent, sel_bounds)

	// The view's coordinate space is in points, while the recorded region is in native
	// pixels. Scale by the same factor the UI derives its screen size with.
	scale := ui.Get().Monitor().DeviceScaleFactor()
	full := image.Rect(0, 0, int(bounds.size.width*scale), int(bounds.size.height*scale))
	keyboard := image.Rect(
		int(viewFrame.origin.x*scale),
		int(viewFrame.origin.y*scale),
		int((viewFrame.origin.x+viewFrame.size.width)*scale),
		int((viewFrame.origin.y+viewFrame.size.height)*scale),
	).Intersect(full)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.vkKnown = true
	if keyboard.Empty() {
		t.vkVisible = false
		t.vkVisibleRegion = full
		return
	}
	t.vkVisible = true
	// An undocked keyboard (an iPad floating or split keyboard) is narrower than
	// the view, so it hides no full-width strip of it. Report the whole view as
	// visible, matching UIKit's own keyboard layout guide.
	if viewFrame.size.width < bounds.size.width {
		t.vkVisibleRegion = full
		return
	}
	t.vkVisibleRegion = image.Rect(0, 0, full.Max.X, min(max(keyboard.Min.Y, 0), full.Max.Y))
}

// readVirtualKeyboard reports whether a virtual keyboard is shown and the
// client-area region it leaves visible, in native pixels. ok is false when the
// region is unknown.
func readVirtualKeyboard() (visible bool, visibleClientRegion image.Rectangle, ok bool) {
	t := &theTextInputImpl
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.vkKnown {
		return false, image.Rectangle{}, false
	}
	return t.vkVisible, t.vkVisibleRegion, true
}
