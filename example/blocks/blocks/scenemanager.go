// Copyright 2014 Hajime Hoshi
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

package blocks

import (
	"github.com/hajimehoshi/ebiten"
)

var (
	transitionFrom *ebiten.Image
	transitionTo   *ebiten.Image
)

func init() {
	var err error
	transitionFrom, err = ebiten.NewImage(ScreenWidth, ScreenHeight, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	transitionTo, err = ebiten.NewImage(ScreenWidth, ScreenHeight, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
}

type Scene interface {
	Update(state *GameState)
	Draw(r *ebiten.Image, images *Images)
}

const transitionMaxCount = 20

type SceneManager struct {
	current         Scene
	next            Scene
	transitionCount int
}

func NewSceneManager(initScene Scene) *SceneManager {
	return &SceneManager{
		current:         initScene,
		transitionCount: -1,
	}
}

func (s *SceneManager) Update(state *GameState) {
	if s.transitionCount == -1 {
		s.current.Update(state)
		return
	}
	s.transitionCount++
	if transitionMaxCount <= s.transitionCount {
		s.current = s.next
		s.next = nil
		s.transitionCount = -1
	}
}

func (s *SceneManager) Draw(r *ebiten.Image, images *Images) {
	if s.transitionCount == -1 {
		s.current.Draw(r, images)
		return
	}
	transitionFrom.Clear()
	s.current.Draw(transitionFrom, images)

	transitionTo.Clear()
	s.next.Draw(transitionTo, images)

	r.DrawImage(transitionFrom, nil)

	alpha := float64(s.transitionCount) / float64(transitionMaxCount)
	r.DrawImage(transitionTo, &ebiten.DrawImageOptions{
		ColorM: ebiten.ScaleColor(1, 1, 1, alpha),
	})
}

func (s *SceneManager) GoTo(scene Scene) {
	s.next = scene
	s.transitionCount = 0
}
