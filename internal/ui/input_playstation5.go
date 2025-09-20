// Copyright 2023 The Ebitengine Authors
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

//go:build playstation5

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

func (u *UserInterface) updateInputStateForFrame() error {
	var err error
	u.mainThread.Call(func() {
		err = u.updateInputStateForFrameImpl()
	})
	return err
}

// updateInputStateForFrameImpl must be called from the main thread.
func (u *UserInterface) updateInputStateForFrameImpl() error {
	if err := gamepad.Update(); err != nil {
		return err
	}
	return nil
}

func (u *UserInterface) KeyName(key Key) string {
	return ""
}
