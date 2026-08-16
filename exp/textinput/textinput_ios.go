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
// dispatched seeding describe the previous target and are dropped.

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
	sel_addObserver                 = objc.RegisterName("addObserver:selector:name:object:")
	sel_addSubview                  = objc.RegisterName("addSubview:")
	sel_alloc                       = objc.RegisterName("alloc")
	sel_applyDismiss                = objc.RegisterName("ebitengineApplyDismiss")
	sel_applyStart                  = objc.RegisterName("ebitengineApplyStart")
	sel_becomeFirstResponder        = objc.RegisterName("becomeFirstResponder")
	sel_bounds                      = objc.RegisterName("bounds")
	sel_convertRectFromView         = objc.RegisterName("convertRect:fromView:")
	sel_convertRectFromWindow       = objc.RegisterName("convertRect:fromWindow:")
	sel_defaultCenter               = objc.RegisterName("defaultCenter")
	sel_init                        = objc.RegisterName("init")
	sel_initWithUTF8String          = objc.RegisterName("initWithUTF8String:")
	sel_keyWindow                   = objc.RegisterName("keyWindow")
	sel_keyboardWillChangeFrame     = objc.RegisterName("ebitengineKeyboardWillChangeFrame:")
	sel_markedTextRange             = objc.RegisterName("markedTextRange")
	sel_objectForKey                = objc.RegisterName("objectForKey:")
	sel_performSelectorOnMainThread = objc.RegisterName("performSelectorOnMainThread:withObject:waitUntilDone:")
	sel_release                     = objc.RegisterName("release")
	sel_resignFirstResponder        = objc.RegisterName("resignFirstResponder")
	sel_selectedRange               = objc.RegisterName("selectedRange")
	sel_setAutocapitalizationType   = objc.RegisterName("setAutocapitalizationType:")
	sel_setDelegate                 = objc.RegisterName("setDelegate:")
	sel_setFrame                    = objc.RegisterName("setFrame:")
	sel_setSelectedRange            = objc.RegisterName("setSelectedRange:")
	sel_setText                     = objc.RegisterName("setText:")
	sel_sharedApplication           = objc.RegisterName("sharedApplication")
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

	// active reports whether text inputting has been requested and not
	// dismissed. closedTicks counts consecutive ticks without text inputting;
	// see dismissVirtualKeyboardIfNeeded.
	active      bool
	closedTicks int

	// legacyCleared drops the delegate events fired by the legacy path
	// clearing the text view.
	legacyCleared bool

	// vkVisible and vkVisibleRegion are the virtual keyboard state reported by
	// UIKit, written on the main thread. vkKnown reports whether a keyboard
	// notification has been observed.
	vkVisible       bool
	vkVisibleRegion image.Rectangle
	vkKnown         bool

	// textView, delegate, and parentView are used only on the main thread,
	// after ensureUIKit.
	textView   objc.ID
	delegate   objc.ID
	parentView objc.ID

	ensureUIKitOnce sync.Once

	mu sync.Mutex
}

var class_EbitengineTextInputDelegate objc.Class

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

		t.delegate = objc.ID(class_EbitengineTextInputDelegate).Send(sel_alloc).Send(sel_init)

		// The keyboard notification names are the constants' string values.
		// The name strings are kept alive for the notification center.
		center := objc.ID(objc.GetClass("NSNotificationCenter")).Send(sel_defaultCenter)
		center.Send(sel_addObserver, t.delegate, sel_keyboardWillChangeFrame, newNSString("UIKeyboardWillChangeFrameNotification"), objc.ID(0))
	})
}

// resolveParentViewOnMain returns the view the text view is attached to: the
// view reported through SetUIView, or the key window as a fallback (the OpenGL
// path does not report its view).
func (t *textInputImpl) resolveParentViewOnMain() objc.ID {
	if v := objc.ID(ui.Get().UIView()); v != 0 {
		return v
	}
	app := objc.ID(objc.GetClass("UIApplication")).Send(sel_sharedApplication)
	return app.Send(sel_keyWindow)
}

// ensureTextViewOnMain creates the hidden text view lazily and attaches it to
// parent.
func (t *textInputImpl) ensureTextViewOnMain(parent objc.ID) objc.ID {
	if t.textView == 0 {
		tv := objc.ID(objc.GetClass("UITextView")).Send(sel_alloc).Send(sel_init)
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
	// Nothing to record: ebitengineApplyStart always rewrites the text view's
	// content, which discards a leftover composition, and dismissing clears it.
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
	t.active = true
	t.closedTicks = 0
	t.sender.reset(value)
}

// pendingStart returns the seed recorded by the last Start.
func (t *textInputImpl) pendingStart() (text string, caretInUTF16 int, bounds image.Rectangle, generation int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pendingText, t.pendingCaretInUTF16, t.pendingBounds, t.generation
}

// markApplied records that the seed of generation was applied to the text view.
func (t *textInputImpl) markApplied(generation int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.appliedGeneration = generation
}

// applyStartOnMain seeds the text view with the pending surrounding text and
// makes it the first responder.
func (t *textInputImpl) applyStartOnMain() {
	text, caret, bounds, gen := t.pendingStart()

	parent := t.resolveParentViewOnMain()
	if parent == 0 {
		return
	}
	tv := t.ensureTextViewOnMain(parent)

	// The text view is kept offscreen: an onscreen one would draw over the
	// game, and a hidden one could not become the first responder. The size
	// mirrors the caret bounds.
	tv.Send(sel_setFrame, cgRect{
		origin: cgPoint{x: -10000, y: -10000},
		size:   cgSize{width: float64(max(bounds.Dx(), 1)), height: float64(max(bounds.Dy(), 1))},
	})

	ns := newNSString(text)
	tv.Send(sel_setText, ns)
	ns.Send(sel_release)
	tv.Send(sel_setSelectedRange, nsRange{location: uint(caret), length: 0})
	tv.Send(sel_becomeFirstResponder)

	t.markApplied(gen)
}

// applyDismissOnMain resigns the first responder, dismissing the virtual
// keyboard, and drops the text view's content along with any composition.
func (t *textInputImpl) applyDismissOnMain() {
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
	s := t.events.getActiveSession()

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.appliedGeneration != t.generation {
		// The event predates the seeding requested by the latest Start and
		// describes the previous target.
		return false
	}

	if s == nil {
		// No session means the deprecated Field is inputting; it uses the
		// legacy whole-value path.
		// TODO: Remove this branch once Field is gone; a Composer session is
		// always active otherwise.
		if !t.events.isOpen() {
			// The event has no receiver: it is the teardown noise of a
			// dismissal, not the user's input.
			return false
		}
		if t.legacyCleared {
			t.legacyCleared = false
			if value == "" {
				return false
			}
		}
		t.events.send(textInputState{
			Text:                             value,
			CompositionSelectionStartInBytes: convertUTF16CountToByteCount(value, selStart),
			CompositionSelectionEndInBytes:   convertUTF16CountToByteCount(value, selEnd),
			ReplacementStartInBytes:          noReplacement,
			ReplacementEndInBytes:            noReplacement,
			CommitKind:                       kind,
		})
		if kind.committed() {
			t.events.end()
			t.legacyCleared = true
			return true
		}
		return false
	}

	// The selection does not track the preedit on iOS; see
	// compositionSelectionInBytes.
	t.sender.trySend(s, value, selStart, selEnd, true, kind)
	return false
}

// textViewEndedEditingOnMain ends text inputting when the first responder is
// resigned, e.g. by the user dismissing the virtual keyboard.
func (t *textInputImpl) textViewEndedEditingOnMain() {
	t.events.end()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active = false
	t.closedTicks = 0
}

// keyboardWillChangeFrameOnMain records the virtual keyboard state from a
// keyboard-frame notification.
func (t *textInputImpl) keyboardWillChangeFrameOnMain(notification objc.ID) {
	parent := t.resolveParentViewOnMain()
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

	full := image.Rect(0, 0, int(bounds.size.width), int(bounds.size.height))
	keyboard := image.Rect(
		int(viewFrame.origin.x),
		int(viewFrame.origin.y),
		int(viewFrame.origin.x+viewFrame.size.width),
		int(viewFrame.origin.y+viewFrame.size.height),
	).Intersect(full)
	// An undocked keyboard (an iPad floating or split keyboard) is narrower
	// than the view and is conventionally ignored, matching UIKit's own
	// keyboard layout guide.
	docked := viewFrame.size.width >= bounds.size.width

	t.mu.Lock()
	defer t.mu.Unlock()
	t.vkKnown = true
	if keyboard.Empty() || !docked {
		t.vkVisible = false
		t.vkVisibleRegion = full
		return
	}
	t.vkVisible = true
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
