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

//go:build android || ios

// Package ebitenmobileview offers functions for OpenGL/Metal view of mobiles.
//
// The functions are not intended for public usages.
// There is no guarantee of backward compatibility.
package ebitenmobileview

// #cgo ios LDFLAGS: -framework UIKit -framework GLKit -framework QuartzCore -framework OpenGLES
//
// #include <stdint.h>
import "C"

import (
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type SetGameNotifier interface {
	NotifySetGame()
}

var theState state

type state struct {
	running         bool
	setGameNotifier SetGameNotifier

	m sync.Mutex
}

func (s *state) isRunning() bool {
	s.m.Lock()
	defer s.m.Unlock()
	return s.running
}

func (s *state) run() {
	s.m.Lock()
	s.running = true
	n := s.setGameNotifier
	s.setGameNotifier = nil
	s.m.Unlock()

	if n != nil {
		n.NotifySetGame()
	}
}

func (s *state) setSetGameNotifier(setGameNotifier SetGameNotifier) {
	s.m.Lock()
	r := s.running
	if !r {
		s.setGameNotifier = setGameNotifier
	}
	s.m.Unlock()

	// If SetGame is already called, notify this immediately.
	if r {
		setGameNotifier.NotifySetGame()
	}
}

func SetGame(game ebiten.Game, options *ebiten.RunGameOptions) {
	if theState.isRunning() {
		panic("ebitenmobileview: SetGame cannot be called twice or more")
	}
	ebiten.RunGameWithoutMainLoop(game, options)
	theState.run()
}

func Layout(viewWidth, viewHeight float64) {
	ui.Get().SetOutsideSize(viewWidth, viewHeight)
}

func Update() error {
	// Lock the OS thread since graphics functions (GL) must be called on this thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !theState.isRunning() {
		// start is not called yet, but as update can be called from another thread, it is OK. Just ignore
		// this.
		return nil
	}

	return ui.Get().Update()
}

func Suspend() error {
	return ui.Get().SetForeground(false)
}

func Resume() error {
	return ui.Get().SetForeground(true)
}

func DeviceScale() float64 {
	return ui.Get().Monitor().DeviceScaleFactor()
}

type RenderRequester interface {
	SetExplicitRenderingMode(explicitRendering bool)
	RequestRenderIfNeeded()
}

func SetRenderRequester(renderRequester RenderRequester) {
	ui.Get().SetRenderRequester(renderRequester)
}

func SetSetGameNotifier(setGameNotifier SetGameNotifier) {
	theState.setSetGameNotifier(setGameNotifier)
}
