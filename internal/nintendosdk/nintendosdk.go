// Copyright 2021 The Ebiten Authors
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

//go:build nintendosdk

package nintendosdk

// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
//
// #include <stdint.h>
//
// struct Touch {
//   int id;
//   int x;
//   int y;
// };
//
// // UI
// void EbitenInitializeGame();
// void EbitenGetScreenSize(int* width, int* height);
// void EbitenBeginFrame();
// void EbitenEndFrame();
//
// // Input
// int EbitenGetTouchNum();
// void EbitenGetTouches(struct Touch* touches);
import "C"

type Touch struct {
	ID int
	X  int
	Y  int
}

func InitializeGame() {
	C.EbitenInitializeGame()
}

func ScreenSize() (int, int) {
	var width, height C.int
	C.EbitenGetScreenSize(&width, &height)
	return int(width), int(height)
}

func BeginFrame() {
	C.EbitenBeginFrame()
}

func EndFrame() {
	C.EbitenEndFrame()
}

var cTouches []C.struct_Touch

func AppendTouches(touches []Touch) []Touch {
	n := int(C.EbitenGetTouchNum())
	cTouches = cTouches[:0]
	if cap(cTouches) < n {
		cTouches = append(cTouches, make([]C.struct_Touch, n)...)
	} else {
		cTouches = cTouches[:n]
	}
	if n > 0 {
		C.EbitenGetTouches(&cTouches[0])
	}

	for _, t := range cTouches {
		touches = append(touches, Touch{
			ID: int(t.id),
			X:  int(t.x),
			Y:  int(t.y),
		})
	}
	return touches
}
