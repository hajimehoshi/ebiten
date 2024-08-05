package text

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/packing"
)

type glyphAtlas struct {
	page  *packing.Page
	image *ebiten.Image
}

type glyphImage struct {
	atlas *glyphAtlas
	node  *packing.Node
}

func (i *glyphImage) Image() *ebiten.Image {
	return i.atlas.image.SubImage(i.node.Region()).(*ebiten.Image)
}

func newGlyphAtlas() *glyphAtlas {
	return &glyphAtlas{
		// Note: 128x128 is arbitrary, maybe a better value can be inferred
		// from the font size or something
		page:  packing.NewPage(128, 128, 1024), // TODO: not 1024
		image: ebiten.NewImage(128, 128),
	}
}

func (g *glyphAtlas) NewImage(w, h int) *glyphImage {
	n := g.page.Alloc(w, h)
	pw, ph := g.page.Size()
	if pw > g.image.Bounds().Dx() || ph > g.image.Bounds().Dy() {
		newImage := ebiten.NewImage(pw, ph)
		newImage.DrawImage(g.image, nil)
		g.image = newImage
	}

	return &glyphImage{
		atlas: g,
		node:  n,
	}
}

func (g *glyphAtlas) Free(img *glyphImage) {
	g.page.Free(img.node)
}


type drawRange struct {
	atlas *glyphAtlas
	end   int
}

// drawList stores triangle versions of DrawImage calls when
// all images are sub-images of an atlas.
// Temporary vertices and indices can be re-used after calling
// Flush, so it is more efficient to keep a reference to a drawList
// instead of creating a new one every frame.
type drawList struct {
	ranges []drawRange
	vx     []ebiten.Vertex
	ix     []uint16
}

// drawCommand is the equivalent of the regular DrawImageOptions
// but only including options that will not break batching.
// Filter, Address, Blend and AntiAlias are determined at Flush()
type drawCommand struct {
	Image *glyphImage

	ColorScale ebiten.ColorScale
	GeoM       ebiten.GeoM
}

var rectIndices = [6]uint16{0, 1, 2, 1, 2, 3}

type point struct {
	X, Y float32
}

func pt(x, y float64) point {
	return point{
		X: float32(x),
		Y: float32(y),
	}
}

type rectOpts struct {
	Dsts         [4]point
	SrcX0, SrcY0 float32
	SrcX1, SrcY1 float32
	R, G, B, A   float32
}

// adjustDestinationPixel is the original ebitengine implementation found here:
// https://github.com/hajimehoshi/ebiten/blob/v2.8.0-alpha.1/internal/graphics/vertex.go#L102-L126
func adjustDestinationPixel(x float32) float32 {
	// Avoid the center of the pixel, which is problematic (#929, #1171).
	// Instead, align the vertices with about 1/3 pixels.
	//
	// The intention here is roughly this code:
	//
	//     float32(math.Floor((float64(x)+1.0/6.0)*3) / 3)
	//
	// The actual implementation is more optimized than the above implementation.
	ix := float32(int(x))
	if x < 0 && x != ix {
		ix -= 1
	}
	frac := x - ix
	switch {
	case frac < 3.0/16.0:
		return ix
	case frac < 8.0/16.0:
		return ix + 5.0/16.0
	case frac < 13.0/16.0:
		return ix + 11.0/16.0
	default:
		return ix + 16.0/16.0
	}
}

func appendRectVerticesIndices(vertices []ebiten.Vertex, indices []uint16, index int, opts *rectOpts) ([]ebiten.Vertex, []uint16) {
	sx0, sy0, sx1, sy1 := opts.SrcX0, opts.SrcY0, opts.SrcX1, opts.SrcY1
	r, g, b, a := opts.R, opts.G, opts.B, opts.A
	vertices = append(vertices,
		ebiten.Vertex{
			DstX:   adjustDestinationPixel(opts.Dsts[0].X),
			DstY:   adjustDestinationPixel(opts.Dsts[0].Y),
			SrcX:   sx0,
			SrcY:   sy0,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		ebiten.Vertex{
			DstX:   adjustDestinationPixel(opts.Dsts[1].X),
			DstY:   adjustDestinationPixel(opts.Dsts[1].Y),
			SrcX:   sx1,
			SrcY:   sy0,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		ebiten.Vertex{
			DstX:   adjustDestinationPixel(opts.Dsts[2].X),
			DstY:   adjustDestinationPixel(opts.Dsts[2].Y),
			SrcX:   sx0,
			SrcY:   sy1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		ebiten.Vertex{
			DstX:   adjustDestinationPixel(opts.Dsts[3].X),
			DstY:   adjustDestinationPixel(opts.Dsts[3].Y),
			SrcX:   sx1,
			SrcY:   sy1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
	)

	indiceCursor := uint16(index * 4)
	indices = append(indices,
		rectIndices[0]+indiceCursor,
		rectIndices[1]+indiceCursor,
		rectIndices[2]+indiceCursor,
		rectIndices[3]+indiceCursor,
		rectIndices[4]+indiceCursor,
		rectIndices[5]+indiceCursor,
	)

	return vertices, indices
}

// Add adds DrawImage commands to the DrawList, images from multiple
// atlases can be added but they will break the previous batch bound to
// a different atlas, requiring an additional draw call internally.
// So, it is better to have the maximum of consecutive DrawCommand images
// sharing the same atlas.
func (dl *drawList) Add(commands ...*drawCommand) {
	if len(commands) == 0 {
		return
	}

	var batch *drawRange

	if len(dl.ranges) > 0 {
		batch = &dl.ranges[len(dl.ranges)-1]
	} else {
		dl.ranges = append(dl.ranges, drawRange{
			atlas: commands[0].Image.atlas,
		})
		batch = &dl.ranges[0]
	}
	// Add vertices and indices
	opts := &rectOpts{}
	for _, cmd := range commands {
		if cmd.Image.atlas != batch.atlas {
			dl.ranges = append(dl.ranges, drawRange{
				atlas: cmd.Image.atlas,
			})
			batch = &dl.ranges[len(dl.ranges)-1]
		}

		// Dst attributes
		bounds := cmd.Image.node.Region()
		opts.Dsts[0] = pt(cmd.GeoM.Apply(0, 0))
		opts.Dsts[1] = pt(cmd.GeoM.Apply(
			float64(bounds.Dx()), 0,
		))
		opts.Dsts[2] = pt(cmd.GeoM.Apply(
			0, float64(bounds.Dy()),
		))
		opts.Dsts[3] = pt(cmd.GeoM.Apply(
			float64(bounds.Dx()), float64(bounds.Dy()),
		))

		// Color and source attributes
		opts.R = cmd.ColorScale.R()
		opts.G = cmd.ColorScale.G()
		opts.B = cmd.ColorScale.B()
		opts.A = cmd.ColorScale.A()
		opts.SrcX0 = float32(bounds.Min.X)
		opts.SrcY0 = float32(bounds.Min.Y)
		opts.SrcX1 = float32(bounds.Max.X)
		opts.SrcY1 = float32(bounds.Max.Y)

		dl.vx, dl.ix = appendRectVerticesIndices(
			dl.vx, dl.ix, batch.end, opts,
		)

		batch.end++
	}
}

// DrawOptions are additional options that will be applied to
// all draw commands from the draw list when calling Flush().
type drawOptions struct {
	ColorScaleMode ebiten.ColorScaleMode
	Blend          ebiten.Blend
	Filter         ebiten.Filter
	Address        ebiten.Address
	AntiAlias      bool
}

// Flush executes all the draw commands as the smallest possible
// amount of draw calls, and then clears the list for next uses.
func (dl *drawList) Flush(dst *ebiten.Image, opts *drawOptions) {
	var topts *ebiten.DrawTrianglesOptions
	if opts != nil {
		topts = &ebiten.DrawTrianglesOptions{
			ColorScaleMode: opts.ColorScaleMode,
			Blend:          opts.Blend,
			Filter:         opts.Filter,
			Address:        opts.Address,
			AntiAlias:      opts.AntiAlias,
		}
	}
	index := 0
	for _, r := range dl.ranges {
		dst.DrawTriangles(
			dl.vx[index*4:(index+r.end)*4],
			dl.ix[index*6:(index+r.end)*6],
			r.atlas.image,
			topts,
		)
		index += r.end
	}
	// Clear buffers
	dl.ranges = dl.ranges[:0]
	dl.vx = dl.vx[:0]
	dl.ix = dl.ix[:0]
}