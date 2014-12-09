package ebiten

type UI interface {
	Start(widht, height, scale int, title string) (Canvas, error)
	DoEvents()
	Terminate()
}

// FIXME: rename this
type Drawer2 interface {
	Draw(c GraphicsContext) error
}

type Canvas interface {
	Draw(drawer Drawer2) error
	IsClosed() bool
}
