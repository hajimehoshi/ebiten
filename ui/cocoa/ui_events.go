package cocoa

import (
	"github.com/hajimehoshi/go-ebiten/ui"
)

type uiEvents struct {
	screenSizeUpdated chan ui.ScreenSizeUpdatedEvent // initialized lazily
	inputStateUpdated chan ui.InputStateUpdatedEvent // initialized lazily
}

func (u *uiEvents) ScreenSizeUpdated() <-chan ui.ScreenSizeUpdatedEvent {
	if u.screenSizeUpdated != nil {
		return u.screenSizeUpdated
	}
	u.screenSizeUpdated = make(chan ui.ScreenSizeUpdatedEvent)
	return u.screenSizeUpdated
}

func (u *uiEvents) notifyScreenSizeUpdated(e ui.ScreenSizeUpdatedEvent) {
	if u.screenSizeUpdated == nil {
		return
	}
	go func() {
		u.screenSizeUpdated <- e
	}()
}

func (u *uiEvents) InputStateUpdated() <-chan ui.InputStateUpdatedEvent {
	if u.inputStateUpdated != nil {
		return u.inputStateUpdated
	}
	u.inputStateUpdated = make(chan ui.InputStateUpdatedEvent)
	return u.inputStateUpdated
}

func (u *uiEvents) notifyInputStateUpdated(e ui.InputStateUpdatedEvent) {
	if u.inputStateUpdated == nil {
		return
	}
	go func() {
		u.inputStateUpdated <- e
	}()
}
