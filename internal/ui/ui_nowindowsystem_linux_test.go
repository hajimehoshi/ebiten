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

//go:build !android && !nintendosdk && !playstation5

package ui_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestNoWindowSystemSizeAfterClose(t *testing.T) {
	closed := false
	width, height, err := ui.NoWindowSystemSizeAfterCloseForTesting(640, 480, func() error {
		closed = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Error("display was not closed")
	}
	if width != 640 || height != 480 {
		t.Errorf("screenSize() = %dx%d, want 640x480", width, height)
	}
}
