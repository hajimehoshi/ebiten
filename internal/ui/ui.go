// Copyright 2015 Hajime Hoshi
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

type UserInterface interface {
	Start(width, height int, scale float64, title string) error
	Update() (interface{}, error)
	SwapBuffers() error
	Terminate() error
	ScreenScale() float64
	SetScreenSize(width, height int) bool
	SetScreenScale(scale float64) bool
}
