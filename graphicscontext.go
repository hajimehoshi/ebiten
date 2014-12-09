package ebiten

// A Rect represents a rectangle.
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// A TexturePart represents a part of a texture.
type TexturePart struct {
	LocationX int
	LocationY int
	Source    Rect
}

// A Drawer is the interface that draws itself.
type Drawer interface {
	Draw(parts []TexturePart, geo GeometryMatrix, color ColorMatrix)
}

// DrawWhole draws the whole texture.
func DrawWhole(drawer Drawer, width, height int, geo GeometryMatrix, color ColorMatrix) {
	parts := []TexturePart{
		{0, 0, Rect{0, 0, width, height}},
	}
	drawer.Draw(parts, geo, color)
}

// A Context is the interface that means a context of rendering.
type GraphicsContext interface {
	Clear()
	Fill(r, g, b uint8)
	Texture(id TextureID) Drawer
	RenderTarget(id RenderTargetID) Drawer

	ResetOffscreen()
	SetOffscreen(id RenderTargetID)
}
