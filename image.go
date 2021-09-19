// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ebiten

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
)

// panicOnErrorAtImageAt indicates whether (*Image).At panics on an error or not.
// This value is set only on testing.
var panicOnErrorAtImageAt bool

// Image represents a rectangle set of pixels.
// The pixel format is alpha-premultiplied RGBA.
// Image implements image.Image and draw.Image.
type Image struct {
	// addr holds self to check copying.
	// See strings.Builder for similar examples.
	addr *Image

	mipmap *mipmap.Mipmap

	bounds   image.Rectangle
	original *Image
	screen   bool
}

func (i *Image) copyCheck() {
	if i.addr != i {
		panic("ebiten: illegal use of non-zero Image copied by value")
	}
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	s := i.Bounds().Size()
	return s.X, s.Y
}

func (i *Image) isDisposed() bool {
	return i.mipmap == nil
}

func (i *Image) isSubImage() bool {
	return i.original != nil
}

// Clear resets the pixels of the image into 0.
//
// When the image is disposed, Clear does nothing.
func (i *Image) Clear() {
	i.Fill(color.Transparent)
}

var (
	emptyImage    = NewImage(3, 3)
	emptySubImage = emptyImage.SubImage(image.Rect(1, 1, 2, 2)).(*Image)
)

func init() {
	w, h := emptyImage.Size()
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	// As emptyImage is used at Fill, use ReplacePixels instead.
	emptyImage.ReplacePixels(pix)
}

// Fill fills the image with a solid color.
//
// When the image is disposed, Fill does nothing.
func (i *Image) Fill(clr color.Color) {
	// Use the original size to cover the entire region (#1691).
	// DrawImage automatically clips the rendering region.
	orig := i
	if i.isSubImage() {
		orig = i.original
	}
	w, h := orig.Size()

	op := &DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))

	r, g, b, a := clr.RGBA()
	var rf, gf, bf, af float64
	if a > 0 {
		rf = float64(r) / float64(a)
		gf = float64(g) / float64(a)
		bf = float64(b) / float64(a)
		af = float64(a) / 0xffff
	}
	op.ColorM.Scale(rf, gf, bf, af)
	op.CompositeMode = CompositeModeCopy

	i.DrawImage(emptySubImage, op)
}

func canSkipMipmap(geom GeoM, filter driver.Filter) bool {
	if filter != driver.FilterLinear {
		return true
	}
	return geom.det2x2() >= 0.999
}

// DrawImageOptions represents options for DrawImage.
type DrawImageOptions struct {
	// GeoM is a geometry matrix to draw.
	// The default (zero) value is identity, which draws the image at (0, 0).
	GeoM GeoM

	// ColorM is a color matrix to draw.
	// The default (zero) value is identity, which doesn't change any color.
	ColorM ColorM

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is regular alpha blending.
	CompositeMode CompositeMode

	// Filter is a type of texture filter.
	// The default (zero) value is FilterNearest.
	Filter Filter
}

// DrawImage draws the given image on the image i.
//
// DrawImage accepts the options. For details, see the document of
// DrawImageOptions.
//
// For drawing, the pixels of the argument image at the time of this call is
// adopted. Even if the argument image is mutated after this call, the drawing
// result is never affected.
//
// When the image i is disposed, DrawImage does nothing.
// When the given image img is disposed, DrawImage panics.
//
// When the given image is as same as i, DrawImage panics.
//
// DrawImage works more efficiently as batches
// when the successive calls of DrawImages satisfy the below conditions:
//
//   * All render targets are same (A in A.DrawImage(B, op))
//   * Either all ColorM element values are same or all the ColorM have only
//      diagonal ('scale') elements
//     * If only (*ColorM).Scale is applied to a ColorM, the ColorM has only
//       diagonal elements. The other ColorM functions might modify the other
//       elements.
//   * All CompositeMode values are same
//   * All Filter values are same
//
// Even when all the above conditions are satisfied, multiple draw commands can
// be used in really rare cases. Ebiten images usually share an internal
// automatic texture atlas, but when you consume the atlas, or you create a huge
// image, those images cannot be on the same texture atlas. In this case, draw
// commands are separated. The texture atlas size is 4096x4096 so far. Another
// case is when you use an offscreen as a render source. An offscreen doesn't
// share the texture atlas with high probability.
//
// For more performance tips, see https://ebiten.org/documents/performancetips.html
func (i *Image) DrawImage(img *Image, options *DrawImageOptions) {
	i.copyCheck()

	if img.isDisposed() {
		panic("ebiten: the given image to DrawImage must not be disposed")
	}
	if i.isDisposed() {
		return
	}

	dstBounds := i.Bounds()
	dstRegion := driver.Region{
		X:      float32(dstBounds.Min.X),
		Y:      float32(dstBounds.Min.Y),
		Width:  float32(dstBounds.Dx()),
		Height: float32(dstBounds.Dy()),
	}

	// Calculate vertices before locking because the user can do anything in
	// options.ImageParts interface without deadlock (e.g. Call Image functions).
	if options == nil {
		options = &DrawImageOptions{}
	}

	bounds := img.Bounds()
	mode := driver.CompositeMode(options.CompositeMode)
	filter := driver.Filter(options.Filter)

	a, b, c, d, tx, ty := options.GeoM.elements32()

	sx0 := float32(bounds.Min.X)
	sy0 := float32(bounds.Min.Y)
	sx1 := float32(bounds.Max.X)
	sy1 := float32(bounds.Max.Y)
	vs := graphics.QuadVertices(sx0, sy0, sx1, sy1, a, b, c, d, tx, ty, 1, 1, 1, 1)
	is := graphics.QuadIndices()

	srcs := [graphics.ShaderImageNum]*mipmap.Mipmap{img.mipmap}

	i.mipmap.DrawTriangles(srcs, vs, is, options.ColorM.affineColorM(), mode, filter, driver.AddressUnsafe, dstRegion, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false, canSkipMipmap(options.GeoM, filter))
}

// Vertex represents a vertex passed to DrawTriangles.
type Vertex struct {
	// DstX and DstY represents a point on a destination image.
	DstX float32
	DstY float32

	// SrcX and SrcY represents a point on a source image.
	// Be careful that SrcX/SrcY coordinates are on the image's bounds.
	// This means that a left-upper point of a sub-image might not be (0, 0).
	SrcX float32
	SrcY float32

	// ColorR/ColorG/ColorB/ColorA represents color scaling values.
	// 1 means the original source image color is used.
	// 0 means a transparent color is used.
	ColorR float32
	ColorG float32
	ColorB float32
	ColorA float32
}

// Address represents a sampler address mode.
type Address int

const (
	// AddressUnsafe means there is no guarantee when the texture coodinates are out of range.
	AddressUnsafe Address = Address(driver.AddressUnsafe)

	// AddressClampToZero means that out-of-range texture coordinates return 0 (transparent).
	AddressClampToZero Address = Address(driver.AddressClampToZero)

	// AddressRepeat means that texture coordinates wrap to the other side of the texture.
	AddressRepeat Address = Address(driver.AddressRepeat)
)

// FillRule is the rule whether an overlapped region is rendered with DrawTriangles(Shader).
type FillRule int

const (
	// FillAll indicates all the triangles are rendered regardless of overlaps.
	FillAll FillRule = iota

	// EvenOdd means that triangles are rendered based on the even-odd rule.
	// If and only if the number of overlappings is odd, the region is rendered.
	EvenOdd
)

// DrawTrianglesOptions represents options for DrawTriangles.
type DrawTrianglesOptions struct {
	// ColorM is a color matrix to draw.
	// The default (zero) value is identity, which doesn't change any color.
	// ColorM is applied before vertex color scale is applied.
	//
	// If Shader is not nil, ColorM is ignored.
	ColorM ColorM

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is regular alpha blending.
	CompositeMode CompositeMode

	// Filter is a type of texture filter.
	// The default (zero) value is FilterNearest.
	Filter Filter

	// Address is a sampler address mode.
	// The default (zero) value is AddressUnsafe.
	Address Address

	// FillRule indicates the rule how an overlapped region is rendered.
	//
	// The rule EvenOdd is useful when you want to render a complex polygon.
	// A complex polygon is a non-convex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
	// See examples/vector for actual usages.
	//
	// The default (zero) value is FillAll.
	FillRule FillRule
}

// MaxIndicesNum is the maximum number of indices for DrawTriangles.
const MaxIndicesNum = graphics.IndicesNum

// DrawTriangles draws triangles with the specified vertices and their indices.
//
// If len(indices) is not multiple of 3, DrawTriangles panics.
//
// If len(indices) is more than MaxIndicesNum, DrawTriangles panics.
//
// The rule in which DrawTriangles works effectively is same as DrawImage's.
//
// When the given image is disposed, DrawTriangles panics.
//
// When the image i is disposed, DrawTriangles does nothing.
func (i *Image) DrawTriangles(vertices []Vertex, indices []uint16, img *Image, options *DrawTrianglesOptions) {
	i.copyCheck()

	if img != nil && img.isDisposed() {
		panic("ebiten: the given image to DrawTriangles must not be disposed")
	}
	if i.isDisposed() {
		return
	}

	if len(indices)%3 != 0 {
		panic("ebiten: len(indices) % 3 must be 0")
	}
	if len(indices) > MaxIndicesNum {
		panic("ebiten: len(indices) must be <= MaxIndicesNum")
	}
	// TODO: Check the maximum value of indices and len(vertices)?

	dstBounds := i.Bounds()
	dstRegion := driver.Region{
		X:      float32(dstBounds.Min.X),
		Y:      float32(dstBounds.Min.Y),
		Width:  float32(dstBounds.Dx()),
		Height: float32(dstBounds.Dy()),
	}

	if options == nil {
		options = &DrawTrianglesOptions{}
	}

	mode := driver.CompositeMode(options.CompositeMode)

	address := driver.Address(options.Address)
	var sr driver.Region
	if address != driver.AddressUnsafe {
		b := img.Bounds()
		sr = driver.Region{
			X:      float32(b.Min.X),
			Y:      float32(b.Min.Y),
			Width:  float32(b.Dx()),
			Height: float32(b.Dy()),
		}
	}

	filter := driver.Filter(options.Filter)

	vs := graphics.Vertices(len(vertices))
	for i, v := range vertices {
		vs[i*graphics.VertexFloatNum] = v.DstX
		vs[i*graphics.VertexFloatNum+1] = v.DstY
		vs[i*graphics.VertexFloatNum+2] = v.SrcX
		vs[i*graphics.VertexFloatNum+3] = v.SrcY
		vs[i*graphics.VertexFloatNum+4] = v.ColorR
		vs[i*graphics.VertexFloatNum+5] = v.ColorG
		vs[i*graphics.VertexFloatNum+6] = v.ColorB
		vs[i*graphics.VertexFloatNum+7] = v.ColorA
	}
	is := make([]uint16, len(indices))
	copy(is, indices)

	srcs := [graphics.ShaderImageNum]*mipmap.Mipmap{img.mipmap}

	i.mipmap.DrawTriangles(srcs, vs, is, options.ColorM.affineColorM(), mode, filter, address, dstRegion, sr, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, options.FillRule == EvenOdd, false)
}

// DrawTrianglesShaderOptions represents options for DrawTrianglesShader.
//
// This API is experimental.
type DrawTrianglesShaderOptions struct {
	// CompositeMode is a composite mode to draw.
	// The default (zero) value is regular alpha blending.
	CompositeMode CompositeMode

	// Uniforms is a set of uniform variables for the shader.
	// The keys are the names of the uniform variables.
	// The values must be float or []float.
	// If the uniform variable type is an array, a vector or a matrix,
	// you have to specify linearly flattened values as a slice.
	// For example, if the uniform variable type is [4]vec4, the number of the slice values will be 16.
	Uniforms map[string]interface{}

	// Images is a set of the source images.
	// All the image must be the same size.
	Images [4]*Image

	// FillRule indicates the rule how an overlapped region is rendered.
	//
	// The rule EvenOdd is useful when you want to render a complex polygon.
	// A complex polygon is a non-convex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
	// See examples/vector for actual usages.
	//
	// The default (zero) value is FillAll.
	FillRule FillRule
}

func init() {
	var op DrawTrianglesShaderOptions
	if got, want := len(op.Images), graphics.ShaderImageNum; got != want {
		panic(fmt.Sprintf("ebiten: len((DrawTrianglesShaderOptions{}).Images) must be %d but %d", want, got))
	}
}

// DrawTrianglesShader draws triangles with the specified vertices and their indices with the specified shader.
//
// For the details about the shader, see https://ebiten.org/documents/shader.html.
//
// If len(indices) is not multiple of 3, DrawTrianglesShader panics.
//
// If len(indices) is more than MaxIndicesNum, DrawTrianglesShader panics.
//
// When a specified image is non-nil and is disposed, DrawTrianglesShader panics.
//
// When the image i is disposed, DrawTrianglesShader does nothing.
//
// This API is experimental.
func (i *Image) DrawTrianglesShader(vertices []Vertex, indices []uint16, shader *Shader, options *DrawTrianglesShaderOptions) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	if len(indices)%3 != 0 {
		panic("ebiten: len(indices) % 3 must be 0")
	}
	if len(indices) > MaxIndicesNum {
		panic("ebiten: len(indices) must be <= MaxIndicesNum")
	}
	// TODO: Check the maximum value of indices and len(vertices)?

	dstBounds := i.Bounds()
	dstRegion := driver.Region{
		X:      float32(dstBounds.Min.X),
		Y:      float32(dstBounds.Min.Y),
		Width:  float32(dstBounds.Dx()),
		Height: float32(dstBounds.Dy()),
	}

	if options == nil {
		options = &DrawTrianglesShaderOptions{}
	}

	mode := driver.CompositeMode(options.CompositeMode)

	vs := graphics.Vertices(len(vertices))
	for i, v := range vertices {
		vs[i*graphics.VertexFloatNum] = v.DstX
		vs[i*graphics.VertexFloatNum+1] = v.DstY
		vs[i*graphics.VertexFloatNum+2] = v.SrcX
		vs[i*graphics.VertexFloatNum+3] = v.SrcY
		vs[i*graphics.VertexFloatNum+4] = v.ColorR
		vs[i*graphics.VertexFloatNum+5] = v.ColorG
		vs[i*graphics.VertexFloatNum+6] = v.ColorB
		vs[i*graphics.VertexFloatNum+7] = v.ColorA
	}
	is := make([]uint16, len(indices))
	copy(is, indices)

	var imgs [graphics.ShaderImageNum]*mipmap.Mipmap
	var imgw, imgh int
	for i, img := range options.Images {
		if img == nil {
			continue
		}
		if img.isDisposed() {
			panic("ebiten: the given image to DrawRectShader must not be disposed")
		}
		if i == 0 {
			imgw, imgh = img.Size()
		} else {
			// TODO: Check imgw > 0 && imgh > 0
			if w, h := img.Size(); imgw != w || imgh != h {
				panic("ebiten: all the source images must be the same size with the rectangle")
			}
		}
		imgs[i] = img.mipmap
	}

	var sx, sy float32
	if options.Images[0] != nil {
		b := options.Images[0].Bounds()
		sx = float32(b.Min.X)
		sy = float32(b.Min.Y)
	}

	var sr driver.Region
	if img := options.Images[0]; img != nil {
		b := img.Bounds()
		sr = driver.Region{
			X:      float32(b.Min.X),
			Y:      float32(b.Min.Y),
			Width:  float32(b.Dx()),
			Height: float32(b.Dy()),
		}
	}

	var offsets [graphics.ShaderImageNum - 1][2]float32
	for i, img := range options.Images[1:] {
		if img == nil {
			continue
		}
		b := img.Bounds()
		offsets[i][0] = -sx + float32(b.Min.X)
		offsets[i][1] = -sy + float32(b.Min.Y)
	}

	us := shader.convertUniforms(options.Uniforms)

	i.mipmap.DrawTriangles(imgs, vs, is, affine.ColorMIdentity{}, mode, driver.FilterNearest, driver.AddressUnsafe, dstRegion, sr, offsets, shader.shader, us, options.FillRule == EvenOdd, false)
}

// DrawRectShaderOptions represents options for DrawRectShader.
//
// This API is experimental.
type DrawRectShaderOptions struct {
	// GeoM is a geometry matrix to draw.
	// The default (zero) value is identity, which draws the rectangle at (0, 0).
	GeoM GeoM

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is regular alpha blending.
	CompositeMode CompositeMode

	// Uniforms is a set of uniform variables for the shader.
	// The keys are the names of the uniform variables.
	// The values must be float or []float.
	// If the uniform variable type is an array, a vector or a matrix,
	// you have to specify linearly flattened values as a slice.
	// For example, if the uniform variable type is [4]vec4, the number of the slice values will be 16.
	Uniforms map[string]interface{}

	// Images is a set of the source images.
	// All the image must be the same size with the rectangle.
	Images [4]*Image
}

func init() {
	var op DrawRectShaderOptions
	if got, want := len(op.Images), graphics.ShaderImageNum; got != want {
		panic(fmt.Sprintf("ebiten: len((DrawRectShaderOptions{}).Images) must be %d but %d", want, got))
	}
}

// DrawRectShader draws a rectangle with the specified width and height with the specified shader.
//
// For the details about the shader, see https://ebiten.org/documents/shader.html.
//
// When one of the specified image is non-nil and is disposed, DrawRectShader panics.
//
// When the image i is disposed, DrawRectShader does nothing.
//
// This API is experimental.
func (i *Image) DrawRectShader(width, height int, shader *Shader, options *DrawRectShaderOptions) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	dstBounds := i.Bounds()
	dstRegion := driver.Region{
		X:      float32(dstBounds.Min.X),
		Y:      float32(dstBounds.Min.Y),
		Width:  float32(dstBounds.Dx()),
		Height: float32(dstBounds.Dy()),
	}

	if options == nil {
		options = &DrawRectShaderOptions{}
	}

	mode := driver.CompositeMode(options.CompositeMode)

	var imgs [graphics.ShaderImageNum]*mipmap.Mipmap
	for i, img := range options.Images {
		if img == nil {
			continue
		}
		if img.isDisposed() {
			panic("ebiten: the given image to DrawRectShader must not be disposed")
		}
		if w, h := img.Size(); width != w || height != h {
			panic("ebiten: all the source images must be the same size with the rectangle")
		}
		imgs[i] = img.mipmap
	}

	var sx, sy float32
	if options.Images[0] != nil {
		b := options.Images[0].Bounds()
		sx = float32(b.Min.X)
		sy = float32(b.Min.Y)
	}

	a, b, c, d, tx, ty := options.GeoM.elements32()
	vs := graphics.QuadVertices(sx, sy, sx+float32(width), sy+float32(height), a, b, c, d, tx, ty, 1, 1, 1, 1)
	is := graphics.QuadIndices()

	var sr driver.Region
	if img := options.Images[0]; img != nil {
		b := img.Bounds()
		sr = driver.Region{
			X:      float32(b.Min.X),
			Y:      float32(b.Min.Y),
			Width:  float32(b.Dx()),
			Height: float32(b.Dy()),
		}
	}

	var offsets [graphics.ShaderImageNum - 1][2]float32
	for i, img := range options.Images[1:] {
		if img == nil {
			continue
		}
		b := img.Bounds()
		offsets[i][0] = -sx + float32(b.Min.X)
		offsets[i][1] = -sy + float32(b.Min.Y)
	}

	us := shader.convertUniforms(options.Uniforms)
	i.mipmap.DrawTriangles(imgs, vs, is, affine.ColorMIdentity{}, mode, driver.FilterNearest, driver.AddressUnsafe, dstRegion, sr, offsets, shader.shader, us, false, canSkipMipmap(options.GeoM, driver.FilterNearest))
}

// SubImage returns an image representing the portion of the image p visible through r.
// The returned value shares pixels with the original image.
//
// The returned value is always *ebiten.Image.
//
// If the image is disposed, SubImage returns nil.
//
// A sub-image returned by SubImage can be used as a rendering source and a rendering destination.
// If a sub-image is used as a rendering source, the image is used as if it is a small image.
// If a sub-image is used as a rendering destination, the region being rendered is clipped.
func (i *Image) SubImage(r image.Rectangle) image.Image {
	i.copyCheck()
	if i.isDisposed() {
		return nil
	}

	r = r.Intersect(i.Bounds())
	// Need to check Empty explicitly. See the standard image package implementations.
	if r.Empty() {
		r = image.ZR
	}

	// Keep the original image's reference not to dispose that by GC.
	var orig = i
	if i.isSubImage() {
		orig = i.original
	}

	img := &Image{
		mipmap:   i.mipmap,
		bounds:   r,
		original: orig,
	}
	img.addr = img

	return img
}

// Bounds returns the bounds of the image.
func (i *Image) Bounds() image.Rectangle {
	if i.isDisposed() {
		panic("ebiten: the image is already disposed")
	}
	return i.bounds
}

// ColorModel returns the color model of the image.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// At returns the color of the image at (x, y).
//
// At loads pixels from GPU to system memory if necessary, which means that At can be slow.
//
// At always returns a transparent color if the image is disposed.
//
// Note that an important logic should not rely on values returned by At, since
// the returned values can include very slight differences between some machines.
//
// At can't be called outside the main loop (ebiten.Run's updating function) starts.
func (i *Image) At(x, y int) color.Color {
	r, g, b, a := i.at(x, y)
	return color.RGBA{r, g, b, a}
}

// RGBA64At implements image.RGBA64Image's RGBA64At.
//
// RGBA64At loads pixels from GPU to system memory if necessary, which means
// that RGBA64At can be slow.
//
// RGBA64At always returns a transparent color if the image is disposed.
//
// Note that an important logic should not rely on values returned by RGBA64At,
// since the returned values can include very slight differences between some machines.
//
// RGBA64At can't be called outside the main loop (ebiten.Run's updating function) starts.
func (i *Image) RGBA64At(x, y int) color.RGBA64 {
	r, g, b, a := i.at(x, y)
	return color.RGBA64{uint16(r) * 0x101, uint16(g) * 0x101, uint16(b) * 0x101, uint16(a) * 0x101}
}

func (i *Image) at(x, y int) (r, g, b, a uint8) {
	if i.isDisposed() {
		return 0, 0, 0, 0
	}
	if !image.Pt(x, y).In(i.Bounds()) {
		return 0, 0, 0, 0
	}
	pix, err := i.mipmap.Pixels(x, y, 1, 1)
	if err != nil {
		if panicOnErrorAtImageAt {
			panic(err)
		}
		theUIContext.setError(err)
		return 0, 0, 0, 0
	}
	return pix[0], pix[1], pix[2], pix[3]
}

// Set sets the color at (x, y).
//
// Set loads pixels from GPU to system memory if necessary, which means that Set can be slow.
//
// In the current implementation, successive calls of Set invokes loading pixels at most once, so this is efficient.
//
// If the image is disposed, Set does nothing.
func (i *Image) Set(x, y int, clr color.Color) {
	i.copyCheck()
	if i.isDisposed() {
		return
	}
	if !image.Pt(x, y).In(i.Bounds()) {
		return
	}
	if i.isSubImage() {
		i = i.original
	}

	r, g, b, a := clr.RGBA()
	pix := []byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)}
	if err := i.mipmap.ReplacePixels(pix, x, y, 1, 1); err != nil {
		theUIContext.setError(err)
	}
}

// Dispose disposes the image data.
// After disposing, most of image functions do nothing and returns meaningless values.
//
// Calling Dispose is not mandatory. GC automatically collects internal resources that no objects refer to.
// However, calling Dispose explicitly is helpful if memory usage matters.
//
// If the image is a sub-image, Dispose does nothing.
//
// When the image is disposed, Dipose does nothing.
func (i *Image) Dispose() {
	i.copyCheck()

	if i.isDisposed() {
		return
	}
	if i.isSubImage() {
		return
	}
	i.mipmap.MarkDisposed()
	i.mipmap = nil
}

// ReplacePixels replaces the pixels of the image with p.
//
// The given p must represent RGBA pre-multiplied alpha values.
// len(pix) must equal to 4 * (bounds width) * (bounds height).
//
// ReplacePixels works on a sub-image.
//
// When len(pix) is not appropriate, ReplacePixels panics.
//
// When the image is disposed, ReplacePixels does nothing.
func (i *Image) ReplacePixels(pixels []byte) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}
	r := i.Bounds()

	// Do not need to copy pixels here.
	// * In internal/mipmap, pixels are copied when necessary.
	// * In internal/shareable, pixels are copied to make its paddings.
	if err := i.mipmap.ReplacePixels(pixels, r.Min.X, r.Min.Y, r.Dx(), r.Dy()); err != nil {
		theUIContext.setError(err)
	}
}

// NewImage returns an empty image.
//
// If width or height is less than 1 or more than device-dependent maximum size, NewImage panics.
//
// NewImage panics if RunGame already finishes.
func NewImage(width, height int) *Image {
	if isRunGameEnded() {
		panic(fmt.Sprintf("ebiten: NewImage cannot be called after RunGame finishes"))
	}
	if width <= 0 {
		panic(fmt.Sprintf("ebiten: width at NewImage must be positive but %d", width))
	}
	if height <= 0 {
		panic(fmt.Sprintf("ebiten: height at NewImage must be positive but %d", height))
	}
	i := &Image{
		mipmap: mipmap.New(width, height),
		bounds: image.Rect(0, 0, width, height),
	}
	i.addr = i
	return i
}

// NewImageFromImage creates a new image with the given image (source).
//
// If source's width or height is less than 1 or more than device-dependent maximum size, NewImageFromImage panics.
//
// NewImageFromImage panics if RunGame already finishes.
func NewImageFromImage(source image.Image) *Image {
	if isRunGameEnded() {
		panic(fmt.Sprintf("ebiten: NewImage cannot be called after RunGame finishes"))
	}

	size := source.Bounds().Size()
	width, height := size.X, size.Y
	if width <= 0 {
		panic(fmt.Sprintf("ebiten: source width at NewImageFromImage must be positive but %d", width))
	}
	if height <= 0 {
		panic(fmt.Sprintf("ebiten: source height at NewImageFromImage must be positive but %d", height))
	}

	i := &Image{
		mipmap: mipmap.New(width, height),
		bounds: image.Rect(0, 0, width, height),
	}
	i.addr = i

	i.ReplacePixels(imageToBytes(source))
	return i
}

func newScreenFramebufferImage(width, height int) *Image {
	i := &Image{
		mipmap: mipmap.NewScreenFramebufferMipmap(width, height),
		bounds: image.Rect(0, 0, width, height),
		screen: true,
	}
	i.addr = i
	return i
}
