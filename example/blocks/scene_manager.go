package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

func init() {
	renderTargetSizes["scene_manager_transition_from"] =
		Size{ScreenWidth, ScreenHeight}
	renderTargetSizes["scene_manager_transition_to"] =
		Size{ScreenWidth, ScreenHeight}
}

type Scene interface {
	Update(state *GameState)
	Draw(context graphics.Context)
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

func (s *SceneManager) Draw(context graphics.Context) {
	if s.transitionCount == -1 {
		s.current.Draw(context)
		return
	}
	from := drawInfo.renderTargets["scene_manager_transition_from"]
	context.SetOffscreen(from)
	context.Clear()
	s.current.Draw(context)

	to := drawInfo.renderTargets["scene_manager_transition_to"]
	context.SetOffscreen(to)
	context.Clear()
	s.next.Draw(context)

	context.ResetOffscreen()
	color := matrix.IdentityColor()
	context.RenderTarget(from).Draw(matrix.IdentityGeometry(), color)

	alpha := float64(s.transitionCount) / float64(transitionMaxCount)
	color.Elements[3][3] = alpha
	context.RenderTarget(to).Draw(matrix.IdentityGeometry(), color)
}

func (s *SceneManager) GoTo(scene Scene) {
	s.next = scene
	s.transitionCount = 0
}
