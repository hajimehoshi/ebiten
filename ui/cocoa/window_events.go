package cocoa

import (
	"github.com/hajimehoshi/go-ebiten/ui"
)

type windowEvents struct {
	screenSizeUpdated chan ui.ScreenSizeUpdatedEvent // initialized lazily
	mouseStateUpdated chan ui.MouseStateUpdatedEvent // initialized lazily
	windowClosed      chan ui.WindowClosedEvent      // initialized lazily
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

func (w *windowEvents) MouseStateUpdated() <-chan ui.MouseStateUpdatedEvent {
	if w.mouseStateUpdated != nil {
		return w.mouseStateUpdated
	}
	w.mouseStateUpdated = make(chan ui.MouseStateUpdatedEvent)
	return w.mouseStateUpdated
}

func (w *windowEvents) notifyInputStateUpdated(e ui.MouseStateUpdatedEvent) {
	if w.mouseStateUpdated == nil {
		return
	}
	go func() {
		w.mouseStateUpdated <- e
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
