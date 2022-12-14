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
	"sync"
	"sync/atomic"
)

var theGlobalState = globalState{
	isScreenClearedEveryFrame_: 1,
	graphicsLibrary_:           int32(GraphicsLibraryUnknown),
}

// globalState represents a global state in this package.
// This is available even before the game loop starts.
type globalState struct {
	err_ error
	errM sync.Mutex

	fpsMode_                   int32
	isScreenClearedEveryFrame_ int32
	graphicsLibrary_           int32
}

func (g *globalState) error() error {
	g.errM.Lock()
	defer g.errM.Unlock()
	return g.err_
}

func (g *globalState) setError(err error) {
	g.errM.Lock()
	defer g.errM.Unlock()
	if g.err_ == nil {
		g.err_ = err
	}
}

func (g *globalState) fpsMode() FPSModeType {
	return FPSModeType(atomic.LoadInt32(&g.fpsMode_))
}

func (g *globalState) setFPSMode(fpsMode FPSModeType) {
	atomic.StoreInt32(&g.fpsMode_, int32(fpsMode))
}

func (g *globalState) isScreenClearedEveryFrame() bool {
	return atomic.LoadInt32(&g.isScreenClearedEveryFrame_) != 0
}

func (g *globalState) setScreenClearedEveryFrame(cleared bool) {
	v := int32(0)
	if cleared {
		v = 1
	}
	atomic.StoreInt32(&g.isScreenClearedEveryFrame_, v)
}

func (g *globalState) setGraphicsLibrary(library GraphicsLibrary) {
	atomic.StoreInt32(&g.graphicsLibrary_, int32(library))
}

func (g *globalState) graphicsLibrary() GraphicsLibrary {
	return GraphicsLibrary(atomic.LoadInt32(&g.graphicsLibrary_))
}

func FPSMode() FPSModeType {
	return theGlobalState.fpsMode()
}

func SetFPSMode(fpsMode FPSModeType) {
	theGlobalState.setFPSMode(fpsMode)
	theUI.SetFPSMode(fpsMode)
}

func IsScreenClearedEveryFrame() bool {
	return theGlobalState.isScreenClearedEveryFrame()
}

func SetScreenClearedEveryFrame(cleared bool) {
	theGlobalState.setScreenClearedEveryFrame(cleared)
}

func GetGraphicsLibrary() GraphicsLibrary {
	return theGlobalState.graphicsLibrary()
}
