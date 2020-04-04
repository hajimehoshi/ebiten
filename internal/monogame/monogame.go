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

// TODO: This implementation depends on some C# files that are not uploaded yet.
// Create 'ebitenmonogame' command to generate C# project for the MonoGame.

// TODO: Update this
const temporaryNamespace = "Go2DotNet.Example.Rotate"

type UpdateDrawer interface {
	Update() error
	Draw() error
}

type Game struct {
	binding js.Value
	update  js.Func
	draw    js.Func
}

var currentGame *Game

func CurrentGame() *Game {
	return currentGame
}

func NewGame(ud UpdateDrawer) *Game {
	update := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return ud.Update()
	})

	draw := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return ud.Draw()
	})

	v := js.Global().Get(".net").Get(temporaryNamespace+".GoBinding").New(update, draw)
	g := &Game{
		binding: v,
		update:  update,
		draw:    draw,
	}
	runtime.SetFinalizer(g, (*Game).Dispose)
	currentGame = g
	return g
}

func (g *Game) Dispose() {
	runtime.SetFinalizer(g, nil)
	g.update.Release()
	g.draw.Release()
	currentGame = nil
}

func (g *Game) Run() {
	g.binding.Call("Run")
}

func (g *Game) NewRenderTarget2D(width, height int) *RenderTarget2D {
	r := &RenderTarget2D{
		v: g.binding.Call("NewRenderTarget2D", width, height),
	}
	runtime.SetFinalizer(r, (*RenderTarget2D).Dispose)
	return r
}

type RenderTarget2D struct {
	v js.Value
}

func (r *RenderTarget2D) Dispose() {
	runtime.SetFinalizer(r, nil)
	r.v.Call("Dispose")
}
