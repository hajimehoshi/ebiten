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

package testing

import (
	"errors"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/testflock"
)

var regularTermination = errors.New("regular termination")

type game struct {
	m    *testing.M
	code int
}

func (g *game) Update() error {
	g.code = g.m.Run()
	return regularTermination
}

func (*game) Draw(*ebiten.Image) {
}

func (*game) Layout(int, int) (int, int) {
	return 320, 240
}

func MainWithRunLoop(m *testing.M) {
	testflock.Lock()
	defer testflock.Unlock()

	// Run an Ebiten process so that (*Image).At is available.
	g := &game{
		m: m,
	}
	if err := ebiten.RunGame(g); err != nil && err != regularTermination {
		panic(err)
	}
	os.Exit(g.code)
}
