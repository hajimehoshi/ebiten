package game

import (
	_ "../graphics"
)

type Game interface {
	Update()
	Draw()
}
