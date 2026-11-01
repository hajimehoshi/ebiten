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

package pinch

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ViewGesture is how the gestures at a tick ask to move, scale, and turn a view.
//
// It is the same whichever device the gestures come from: scrolling, pinching, and rotating on a touchpad
// or with a mouse wheel, or the fingers on a touch screen, where one finger moves the view and two fingers
// move it with their middle point.
type ViewGesture struct {
	// PanX and PanY are how far the view moves, in pixels.
	PanX, PanY float64

	// Scale is the ratio the view scales by around the center. 1 means no change.
	Scale float64

	// Rotation is the angle the view turns by around the center, in radians. A positive value is clockwise.
	Rotation float64

	// CenterX and CenterY are the point the view scales and turns around: the middle of the fingers when
	// two fingers touch the screen, and the cursor otherwise.
	CenterX, CenterY float64
}

// Apply applies the gesture to view, which maps a world to the screen.
func (v *ViewGesture) Apply(view *ebiten.GeoM) {
	view.Translate(v.PanX, v.PanY)
	view.Translate(-v.CenterX, -v.CenterY)
	view.Scale(v.Scale, v.Scale)
	view.Rotate(v.Rotation)
	view.Translate(v.CenterX, v.CenterY)
}

// Gesture is a set of the ways a gesture changes a view.
type Gesture int

const (
	// GesturePanX moves the view horizontally.
	GesturePanX Gesture = 1 << iota

	// GesturePanY moves the view vertically.
	GesturePanY

	// GestureScale scales the view.
	GestureScale

	// GestureRotate turns the view.
	GestureRotate
)

func (g Gesture) String() string {
	var names []string
	for _, n := range []struct {
		g    Gesture
		name string
	}{
		{GesturePanX, "PanX"},
		{GesturePanY, "PanY"},
		{GestureScale, "Scale"},
		{GestureRotate, "Rotate"},
	} {
		if g&n.g != 0 {
			names = append(names, n.name)
		}
	}
	if len(names) == 0 {
		return "None"
	}
	return strings.Join(names, "|")
}

type touchPosition struct {
	x, y float64
}

const (
	// panLockThreshold is how far a gesture moves before it is decided which way it goes, in pixels.
	panLockThreshold = 8

	// panLockRatio is how many times as far a gesture moves along an axis as across it to be locked to the axis.
	// A gesture that moves less than this along both axes moves in any direction.
	panLockRatio = 2

	// transformLockThreshold is how far a gesture scales or turns before it is decided which it does. Scaling is
	// measured as the logarithm of the ratio, and turning in radians, so that both are as far as the fingers move
	// relative to the distance between them.
	transformLockThreshold = 0.05

	// transformLockRatio is 1 so that a gesture always either scales or turns, whichever it does more.
	transformLockRatio = 1

	// gestureGapTicks is the number of ticks without changes that ends a gesture without touches, such as one on
	// a touchpad.
	gestureGapTicks = 10
)

type lockState int

const (
	// lockUndecided is a gesture that has not changed the view enough to tell what it does.
	lockUndecided lockState = iota
	lockA
	lockB
	lockBoth
)

// exclusiveLock keeps a gesture doing only one of two things, a and b: whichever it does first.
type exclusiveLock struct {
	state lockState
	// pendingA and pendingB are what an undecided gesture has done so far.
	pendingA  float64
	pendingB  float64
	idleTicks int
}

// restrict restricts a and b, what a gesture does at the current tick, in units where threshold is as far for
// both. A gesture is locked to one of them when it does ratio times as much of it as of the other.
// touchesChanged reports whether fingers touched or left the screen, which starts a new gesture.
func (l *exclusiveLock) restrict(a, b *float64, threshold, ratio float64, touches int, touchesChanged bool) {
	if *a == 0 && *b == 0 {
		l.idleTicks++
	} else {
		l.idleTicks = 0
	}
	// A gesture on a touch screen lasts while the same fingers touch it, even when they stop moving.
	if touchesChanged || (touches == 0 && l.idleTicks >= gestureGapTicks) {
		*l = exclusiveLock{}
	}

	switch l.state {
	case lockUndecided:
		l.pendingA += *a
		l.pendingB += *b
		x, y := math.Abs(l.pendingA), math.Abs(l.pendingB)
		if max(x, y) < threshold {
			*a, *b = 0, 0
			return
		}
		// Apply what the gesture has done so far, so that the start of it is not lost.
		*a, *b = l.pendingA, l.pendingB
		switch {
		case x >= y*ratio:
			l.state = lockA
			*b = 0
		case y >= x*ratio:
			l.state = lockB
			*a = 0
		default:
			l.state = lockBoth
		}
	case lockA:
		*b = 0
	case lockB:
		*a = 0
	}
}

// GestureTracker makes a ViewGesture at each tick. It remembers the touches of the previous tick to know how
// the fingers moved.
//
// By default, a gesture moves, scales, and turns the view in any combination. Ignore and Separate restrict it.
type GestureTracker struct {
	// Ignore is the gestures not to report. For example, GesturePanX moves the view only vertically, as in a
	// list.
	Ignore Gesture

	// Separate is the gestures a gesture does only one of: whichever it does first. It applies between
	// GesturePanX and GesturePanY, and between GestureScale and GestureRotate, when both of a pair are in it.
	//
	// GesturePanX|GesturePanY keeps a gesture moving along the axis it starts along, as in a document. A
	// gesture that starts diagonally moves the view in any direction.
	//
	// GestureScale|GestureRotate keeps a gesture either scaling or turning, as doing both at once is rarely
	// intended and hard to control.
	Separate Gesture

	touchIDs []ebiten.TouchID
	prev     map[ebiten.TouchID]touchPosition

	panLock       exclusiveLock
	transformLock exclusiveLock
}

// restrict restricts v to t.Ignore and t.Separate.
func (t *GestureTracker) restrict(v *ViewGesture, touches int, touchesChanged bool) {
	if t.Ignore&GesturePanX != 0 {
		v.PanX = 0
	}
	if t.Ignore&GesturePanY != 0 {
		v.PanY = 0
	}
	if t.Ignore&GestureScale != 0 {
		v.Scale = 1
	}
	if t.Ignore&GestureRotate != 0 {
		v.Rotation = 0
	}

	if s := GesturePanX | GesturePanY; t.Separate&s == s {
		t.panLock.restrict(&v.PanX, &v.PanY, panLockThreshold, panLockRatio, touches, touchesChanged)
	}
	if s := GestureScale | GestureRotate; t.Separate&s == s {
		var scale float64
		if v.Scale > 0 {
			scale = math.Log(v.Scale)
		}
		t.transformLock.restrict(&scale, &v.Rotation, transformLockThreshold, transformLockRatio, touches, touchesChanged)
		v.Scale = math.Exp(scale)
	}
}

// Update returns the gesture at the current tick. Call it once at each tick.
func (t *GestureTracker) Update() ViewGesture {
	if t.prev == nil {
		t.prev = map[ebiten.TouchID]touchPosition{}
	}

	v := ViewGesture{
		Scale:    1 + ebiten.PinchDelta(),
		Rotation: ebiten.RotationDelta(),
	}
	v.PanX, v.PanY = ebiten.ScrollDelta()
	v.CenterX, v.CenterY = inpututil.PinchCenter()

	// A pinch and a rotation on a touch screen are already in PinchDelta and RotationDelta on the platforms
	// that report them, around the middle point of the two fingers. The touches add the move: one finger
	// moves the view, and two fingers move it with their middle point.
	t.touchIDs = ebiten.AppendTouchIDs(t.touchIDs[:0])
	if n := len(t.touchIDs); n == 1 || n == 2 {
		var x, y, prevX, prevY float64
		// The fingers move the view only when the same fingers touched the screen at the previous tick.
		same := len(t.prev) == n
		for _, id := range t.touchIDs {
			tx, ty := ebiten.TouchPositionF(id)
			x += tx
			y += ty
			p, ok := t.prev[id]
			if !ok {
				same = false
			}
			prevX += p.x
			prevY += p.y
		}
		x /= float64(n)
		y /= float64(n)
		if same {
			v.PanX += x - prevX/float64(n)
			v.PanY += y - prevY/float64(n)
		}
	}

	touchesChanged := len(t.prev) != len(t.touchIDs)
	for _, id := range t.touchIDs {
		if _, ok := t.prev[id]; !ok {
			touchesChanged = true
		}
	}

	clear(t.prev)
	for _, id := range t.touchIDs {
		x, y := ebiten.TouchPositionF(id)
		t.prev[id] = touchPosition{x: x, y: y}
	}

	t.restrict(&v, len(t.touchIDs), touchesChanged)
	return v
}
