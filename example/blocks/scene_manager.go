package blocks

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
)

type GameState struct {
	SceneManager *SceneManager
}

type Scene interface {
	Update(state GameState)
	Draw(context graphics.Context)
}

type SceneManager struct {
	current Scene
	next    Scene
}

func NewSceneManager(initScene Scene) *SceneManager {
	return &SceneManager{
		current: initScene,
	}
}

func (s *SceneManager) Update() {
	if s.next != nil {
		s.current = s.next
		s.next = nil
	}
	s.current.Update(GameState{
		SceneManager: s,
	})
}

func (s *SceneManager) Draw(context graphics.Context) {
	s.current.Draw(context)
}

func (s *SceneManager) GoTo(scene Scene) {
	s.next = scene
}
