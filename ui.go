package ebiten

type UI interface {
	Start(widht, height, scale int, title string) (Canvas, error)
	DoEvents()
	Terminate()
}

type GraphicsContextDrawer interface {
	Draw(c GraphicsContext) error
}

type Canvas interface {
	Draw(drawer GraphicsContextDrawer) error
	IsClosed() bool
}
