/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package blocks

import (
	"github.com/hajimehoshi/ebiten"
)

func init() {
	renderTargetSizes["scene_manager_transition_from"] =
		Size{ScreenWidth, ScreenHeight}
	renderTargetSizes["scene_manager_transition_to"] =
		Size{ScreenWidth, ScreenHeight}
}

type Scene interface {
	Update(state *GameState)
	Draw(context ebiten.GraphicsContext, textures *Textures)
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

func (s *SceneManager) Draw(context ebiten.GraphicsContext, textures *Textures) {
	if s.transitionCount == -1 {
		s.current.Draw(context, textures)
		return
	}
	from := textures.GetRenderTarget("scene_manager_transition_from")
	context.PushRenderTarget(from)
	context.Clear()
	s.current.Draw(context, textures)
	context.PopRenderTarget()

	to := textures.GetRenderTarget("scene_manager_transition_to")
	context.PushRenderTarget(to)
	context.Clear()
	s.next.Draw(context, textures)
	context.PopRenderTarget()

	color := ebiten.ColorMatrixI()
	ebiten.DrawWholeRenderTarget(context, from, ebiten.GeometryMatrixI(), color)

	alpha := float64(s.transitionCount) / float64(transitionMaxCount)
	color.Elements[3][3] = alpha
	ebiten.DrawWholeRenderTarget(context, to, ebiten.GeometryMatrixI(), color)
}

func (s *SceneManager) GoTo(scene Scene) {
	s.next = scene
	s.transitionCount = 0
}
