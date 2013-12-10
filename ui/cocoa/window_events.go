package cocoa

import (
	"github.com/hajimehoshi/go-ebiten/ui"
)

type windowEvents struct {
	screenSizeUpdated chan ui.ScreenSizeUpdatedEvent // initialized lazily
	inputStateUpdated chan ui.InputStateUpdatedEvent // initialized lazily
	windowClosed chan ui.WindowClosedEvent // initialized lazily
}

func (w *windowEvents) ScreenSizeUpdated() <-chan ui.ScreenSizeUpdatedEvent {
	if w.screenSizeUpdated != nil {
		return w.screenSizeUpdated
	}
	w.screenSizeUpdated = make(chan ui.ScreenSizeUpdatedEvent)
	return w.screenSizeUpdated
}

func (w *windowEvents) notifyScreenSizeUpdated(e ui.ScreenSizeUpdatedEvent) {
	if w.screenSizeUpdated == nil {
		return
	}
	go func() {
		w.screenSizeUpdated <- e
	}()
}

func (w *windowEvents) InputStateUpdated() <-chan ui.InputStateUpdatedEvent {
	if w.inputStateUpdated != nil {
		return w.inputStateUpdated
	}
	w.inputStateUpdated = make(chan ui.InputStateUpdatedEvent)
	return w.inputStateUpdated
}

func (w *windowEvents) notifyInputStateUpdated(e ui.InputStateUpdatedEvent) {
	if w.inputStateUpdated == nil {
		return
	}
	go func() {
		w.inputStateUpdated <- e
	}()
}

func (w *windowEvents) WindowClosed() <-chan ui.WindowClosedEvent {
	if w.windowClosed != nil {
		return w.windowClosed
	}
	w.windowClosed = make(chan ui.WindowClosedEvent)
	return w.windowClosed
}

func (w *windowEvents) notifyWindowClosed(e ui.WindowClosedEvent) {
	if w.windowClosed == nil {
		return
	}
	go func() {
		w.windowClosed <- e
	}()
}

