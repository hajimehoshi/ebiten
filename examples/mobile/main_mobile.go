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

// +build android

// Package mobile is an example of Ebiten for mobiles (Android).
// You can `gomobile bind` this package but not `gomobile build`.
// Please run `gomobile bind` like:
//
// Android:
//   gomobile bind -javapkg [Your package path] -o [AAR file path] github.com/hajimehoshi/ebiten/examples/mobile
// iOS:
//   (TBD)
package mobile

import (
	example "github.com/hajimehoshi/ebiten/examples/mobile/mobile"
	"github.com/hajimehoshi/ebiten/mobile"
)

// EventDispacher must be redeclared and exported so that this is available on the Java/Objective-C side.

type EventDispatcher mobile.EventDispatcher

func Start() (EventDispatcher, error) {
	return mobile.Start(example.Update, example.ScreenWidth, example.ScreenHeight, 2, "Mobile (Ebiten Demo)")
}
