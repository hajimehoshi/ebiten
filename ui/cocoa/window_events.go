package cocoa

import (
	"github.com/hajimehoshi/go-ebiten/ui"
)

type windowEvents struct {
	screenSizeUpdated chan ui.ScreenSizeUpdatedEvent // initialized lazily
	mouseStateUpdated chan ui.MouseStateUpdatedEvent // initialized lazily
	windowClosed      chan ui.WindowClosedEvent      // initialized lazily
}

func (w *windowEvents) notify(e interface{}) {
	go func() {
		w.doNotify(e)
	}()
}

func (w *windowEvents) doNotify(e interface{}) {
	switch e := e.(type) {
	case ui.ScreenSizeUpdatedEvent:
		if w.screenSizeUpdated != nil {
			w.screenSizeUpdated <- e
		}
	case ui.MouseStateUpdatedEvent:
		if w.mouseStateUpdated != nil {
			w.mouseStateUpdated <- e
		}
	case ui.WindowClosedEvent:
		if w.windowClosed != nil {
			w.windowClosed <- e
		}
	}
}

func (w *windowEvents) ScreenSizeUpdated() <-chan ui.ScreenSizeUpdatedEvent {
	if w.screenSizeUpdated != nil {
		return w.screenSizeUpdated
	}
	w.screenSizeUpdated = make(chan ui.ScreenSizeUpdatedEvent)
	return w.screenSizeUpdated
}

func (w *windowEvents) MouseStateUpdated() <-chan ui.MouseStateUpdatedEvent {
	if w.mouseStateUpdated != nil {
		return w.mouseStateUpdated
	}
	w.mouseStateUpdated = make(chan ui.MouseStateUpdatedEvent)
	return w.mouseStateUpdated
}

func (w *windowEvents) WindowClosed() <-chan ui.WindowClosedEvent {
	if w.windowClosed != nil {
		return w.windowClosed
	}
	w.windowClosed = make(chan ui.WindowClosedEvent)
	return w.windowClosed
}
