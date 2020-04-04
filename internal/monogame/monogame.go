// Copyright 2020 The Ebiten Authors
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

package monogame

import (
	"runtime"
	"syscall/js"
)

// TODO: Update this
const temporaryNamespace = "Go2DotNet.Example.Rotate"

type UpdateDrawer interface {
	Update() error
	Draw() error
}

type Game struct {
	v      js.Value
	update js.Func
	draw   js.Func
}

func (g *Game) Run() {
	// Methods named *FromGo is defined to avoid ambiguous matches of methods.
	g.v.Call("RunFromGo")
}

func (g *Game) Dispose() {
	runtime.SetFinalizer(g, nil)
	g.update.Release()
	g.draw.Release()
}

type RenderTarget2D js.Value

func NewGame(ud UpdateDrawer) *Game {
	update := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return ud.Update()
	})

	draw := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return ud.Draw()
	})

	v := js.Global().Get(".net").Get(temporaryNamespace+".GoGame").New(update, draw)
	g := &Game{
		v:      v,
		update: update,
		draw:   draw,
	}
	runtime.SetFinalizer(g, (*Game).Dispose)
	return g
}
