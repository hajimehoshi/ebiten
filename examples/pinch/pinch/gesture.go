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

	// CenterX and CenterY are the point the view scales and turns around, which inpututil.PinchCenter
	// returns.
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
		{
			g:    GesturePanX,
			name: "PanX",
		},
		{
			g:    GesturePanY,
			name: "PanY",
		},
		{
			g:    GestureScale,
			name: "Scale",
		},
		{
			g:    GestureRotate,
			name: "Rotate",
		},
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

// GesturePairs is a set of pairs of the ways a gesture changes a view, which Separate keeps a gesture from doing both of.
type GesturePairs int

const (
	// PairPanXPanY is moving horizontally and moving vertically.
	PairPanXPanY GesturePairs = 1 << iota

	// PairPanScale is moving, in any direction, and scaling.
	PairPanScale

	// PairPanRotate is moving, in any direction, and turning.
	PairPanRotate

	// PairScaleRotate is scaling and turning.
	PairScaleRotate
)

func (p GesturePairs) String() string {
	var names []string
	for _, n := range []struct {
		p    GesturePairs
		name string
	}{
		{
			p:    PairPanXPanY,
			name: "PanX/PanY",
		},
		{
			p:    PairPanScale,
			name: "Pan/Scale",
		},
		{
			p:    PairPanRotate,
			name: "Pan/Rotate",
		},
		{
			p:    PairScaleRotate,
			name: "Scale/Rotate",
		},
	} {
		if p&n.p != 0 {
			names = append(names, n.name)
		}
	}
	if len(names) == 0 {
		return "None"
	}
	return strings.Join(names, "|")
}

// gestureKind is a kind of the ways a gesture changes a view. Moving horizontally and moving vertically are one kind.
type gestureKind int

const (
	kindPan gestureKind = iota
	kindScale
	kindRotate
	kindCount
)

// kindPairs is the pairs of the kinds that GesturePairs keeps a gesture from doing both of.
var kindPairs = []struct {
	pair GesturePairs
	a, b gestureKind
}{
	{
		pair: PairPanScale,
		a:    kindPan,
		b:    kindScale,
	},
	{
		pair: PairPanRotate,
		a:    kindPan,
		b:    kindRotate,
	},
	{
		pair: PairScaleRotate,
		a:    kindScale,
		b:    kindRotate,
	},
}

// The amounts of a gesture at a tick, in the units that kindLock uses.
const (
	amountPanX = iota
	amountPanY
	// amountScale is the logarithm of the ratio the view scales by.
	amountScale
	// amountRotate is the angle the view turns by, in radians.
	amountRotate
	amountCount
)

// amountKinds is the kind of each amount.
var amountKinds = [amountCount]gestureKind{
	amountPanX:   kindPan,
	amountPanY:   kindPan,
	amountScale:  kindScale,
	amountRotate: kindRotate,
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

	// transformLockThreshold is how far a gesture scales or turns before it is decided whether it does. Scaling
	// is measured as the logarithm of the ratio, and turning in radians, so that both are as far as the fingers
	// move relative to the distance between them. For fingers 160 pixels apart, this is as far as
	// panLockThreshold.
	transformLockThreshold = 0.05

	// gestureGapTicks is the number of ticks without changes that ends a gesture without touches, such as one on
	// a touchpad.
	gestureGapTicks = 10
)

// ctrlScrollLogScalePerPixel is the logarithm of the ratio scrolling with the Control key scales the view by per pixel
// of scrolling, when it is reported as a pinch. One notch of a mouse wheel, about 100 pixels, scales the view by 20%.
var ctrlScrollLogScalePerPixel = math.Log(1.2) / 100

type lockState int

const (
	// lockUndecided is a gesture that has not changed the view enough to tell what it does.
	lockUndecided lockState = iota
	lockA
	lockB
	lockBoth
)

// kindLock keeps a gesture from doing both kinds of a pair: it does whichever it does first.
type kindLock struct {
	// allowed and blocked are the kinds that the gesture is decided to do and not to do.
	allowed [kindCount]bool
	blocked [kindCount]bool
	// pending is what the gesture has done so far of the kinds that are not decided yet.
	pending   [amountCount]float64
	idleTicks int
}

// magnitude returns how far the gesture has done kind k so far, relative to the threshold to decide it.
func (l *kindLock) magnitude(k gestureKind) float64 {
	switch k {
	case kindPan:
		return math.Hypot(l.pending[amountPanX], l.pending[amountPanY]) / panLockThreshold
	case kindScale:
		return math.Abs(l.pending[amountScale]) / transformLockThreshold
	case kindRotate:
		return math.Abs(l.pending[amountRotate]) / transformLockThreshold
	}
	return 0
}

// restrict restricts amounts, what a gesture does at the current tick. excludes reports whether a gesture must
// not do both of two kinds. touchesChanged reports whether fingers touched or left the screen, which starts a
// new gesture.
func (l *kindLock) restrict(amounts *[amountCount]float64, excludes func(a, b gestureKind) bool, touches int, touchesChanged bool) {
	if *amounts == [amountCount]float64{} {
		l.idleTicks++
	} else {
		l.idleTicks = 0
	}
	// A gesture on a touch screen lasts while the same fingers touch it, even when they stop moving.
	if touchesChanged || (touches == 0 && l.idleTicks >= gestureGapTicks) {
		*l = kindLock{}
	}

	// Hold what the gesture does of the kinds that are not decided yet.
	for i, a := range amounts {
		k := amountKinds[i]
		switch {
		case l.blocked[k]:
			amounts[i] = 0
		case !l.allowed[k]:
			l.pending[i] += a
			amounts[i] = 0
		}
	}

	// Decide the kinds one by one, the one the gesture has done the most first. A kind that no undecided kind
	// excludes is allowed at once.
	for {
		best := gestureKind(-1)
		var bestMagnitude float64
		for k := range kindCount {
			if l.allowed[k] || l.blocked[k] {
				continue
			}
			m := l.magnitude(k)
			if l.free(k, excludes) {
				m = math.Inf(1)
			}
			if m >= 1 && m > bestMagnitude {
				best, bestMagnitude = k, m
			}
		}
		if best < 0 {
			return
		}
		l.allowed[best] = true
		for k := range kindCount {
			if k != best && excludes(best, k) {
				l.blocked[k] = true
			}
		}
		// Apply what the gesture has done so far, so that the start of it is not lost.
		for i, k := range amountKinds {
			if k == best {
				amounts[i] += l.pending[i]
				l.pending[i] = 0
			}
		}
	}
}

// free reports whether no kind that is not blocked excludes k.
func (l *kindLock) free(k gestureKind, excludes func(a, b gestureKind) bool) bool {
	for o := range kindCount {
		if o != k && !l.blocked[o] && excludes(k, o) {
			return false
		}
	}
	return true
}

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

	// Separate is the pairs of gestures a gesture does only one of: whichever it does first. The pairs combine.
	// For example, PairPanScale|PairPanRotate|PairScaleRotate keeps a gesture either moving, scaling, or
	// turning, and PairPanScale|PairPanRotate keeps it either moving, or scaling and turning.
	//
	// PairPanXPanY keeps a gesture moving along the axis it starts along, as in a document. A gesture that starts
	// diagonally moves the view in any direction. It applies after the other pairs, which take moving in any
	// direction as one.
	//
	// PairScaleRotate keeps a gesture either scaling or turning, as doing both at once is rarely intended and hard
	// to control. PairPanScale keeps the view from moving with the middle point of the fingers while they pinch,
	// as in a map.
	Separate GesturePairs

	// CtrlScrollAsPinch reports scrolling with the Control key held as a pinch around the cursor instead of a
	// move. Scrolling up magnifies the view, and horizontal scrolling with the Control key is dropped.
	//
	// This is a fallback for touchpads whose pinches PinchDelta does not report, such as touchpads that are not
	// precision touchpads on Windows: their drivers report a pinch as scrolling with the Control key, as many
	// applications zoom with it. It works with a mouse wheel as well. Browsers report a pinch on a touchpad
	// with PinchDelta, so this is needed there only for drivers that press the Control key themselves.
	//
	// CtrlScrollAsPinch is false by default, as scrolling with the Control key can no longer be used for
	// anything else, such as moving the view faster.
	CtrlScrollAsPinch bool

	touchIDs []ebiten.TouchID
	prev     map[ebiten.TouchID]touchPosition

	kindLock kindLock
	panLock  exclusiveLock
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

	// An ignored kind never excludes another.
	ignored := [kindCount]bool{
		kindPan:    t.Ignore&(GesturePanX|GesturePanY) == GesturePanX|GesturePanY,
		kindScale:  t.Ignore&GestureScale != 0,
		kindRotate: t.Ignore&GestureRotate != 0,
	}
	excludes := func(a, b gestureKind) bool {
		if ignored[a] || ignored[b] {
			return false
		}
		for _, p := range kindPairs {
			if t.Separate&p.pair != 0 && (p.a == a && p.b == b || p.a == b && p.b == a) {
				return true
			}
		}
		return false
	}
	var scale float64
	if v.Scale > 0 {
		scale = math.Log(v.Scale)
	}
	amounts := [amountCount]float64{
		amountPanX:   v.PanX,
		amountPanY:   v.PanY,
		amountScale:  scale,
		amountRotate: v.Rotation,
	}
	t.kindLock.restrict(&amounts, excludes, touches, touchesChanged)
	v.PanX, v.PanY, v.Rotation = amounts[amountPanX], amounts[amountPanY], amounts[amountRotate]
	if v.Scale > 0 {
		v.Scale = math.Exp(amounts[amountScale])
	}

	if t.Separate&PairPanXPanY != 0 {
		t.panLock.restrict(&v.PanX, &v.PanY, panLockThreshold, panLockRatio, touches, touchesChanged)
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
	if t.CtrlScrollAsPinch && ebiten.IsKeyPressed(ebiten.KeyControl) {
		v.Scale *= math.Exp(v.PanY * ctrlScrollLogScalePerPixel)
		v.PanX, v.PanY = 0, 0
	}
	v.CenterX, v.CenterY = inpututil.PinchCenter()

	// Fingers touching or leaving the screen start a new gesture.
	t.touchIDs = ebiten.AppendTouchIDs(t.touchIDs[:0])
	touchesChanged := len(t.prev) != len(t.touchIDs)
	for _, id := range t.touchIDs {
		if _, ok := t.prev[id]; !ok {
			touchesChanged = true
			break
		}
	}
	if touchesChanged {
		clear(t.prev)
	}

	// A pinch and a rotation on a touch screen are already in PinchDelta and RotationDelta on the platforms
	// that report them, around the middle point of the two fingers. The touches add the move: one finger
	// moves the view, and two fingers move it with their middle point, which moves by the average of how far
	// the fingers move. The fingers move the view only when the same fingers touched the screen at the
	// previous tick.
	var dx, dy float64
	for _, id := range t.touchIDs {
		x, y := ebiten.TouchPositionF(id)
		p := t.prev[id]
		dx += x - p.x
		dy += y - p.y
		t.prev[id] = touchPosition{x: x, y: y}
	}
	if n := len(t.touchIDs); !touchesChanged && (n == 1 || n == 2) {
		v.PanX += dx / float64(n)
		v.PanY += dy / float64(n)
	}

	t.restrict(&v, len(t.touchIDs), touchesChanged)
	return v
}
