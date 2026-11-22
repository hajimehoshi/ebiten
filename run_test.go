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

package ebiten_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRunOptionsDoNotChange(t *testing.T) {
	var options ebiten.RunGameOptions
	ebiten.ToUIRunOptions(&options)
	if options != (ebiten.RunGameOptions{}) {
		t.Errorf("caller options changed: got %+v, want zero value", options)
	}
}
