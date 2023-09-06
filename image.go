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
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Image represents a rectangle set of pixels.
// The pixel format is alpha-premultiplied RGBA.
// Image implements the standard image.Image and draw.Image interfaces.
type Image struct {
	// addr holds self to check copying.
	// See strings.Builder for similar examples.
	addr     *Image
	image    *ui.Image
	original *Image
	bounds   image.Rectangle

	// tmpVertices must not be reused until ui.Image.Draw* is called.
	tmpVertices []float32

	// tmpUniforms must not be reused until ui.Image.Draw* is called.
	tmpUniforms []uint32

	// Do not add a 'buffering' member that are resolved lazily.
	// This tends to forget resolving the buffer easily (#2362).
}

func (i *Image) copyCheck() {
	if i.addr != i {
		panic("ebiten: illegal use of non-zero Image copied by value")
	}
}

// Size returns the size of the image.
//
// Deprecated: as of v2.5. Use Bounds().Dx() and Bounds().Dy() or Bounds().Size() instead.
func (i *Image) Size() (width, height int) {
	s := i.Bounds().Size()
	return s.X, s.Y
}

func (i *Image) isDisposed() bool {
	return i.image == nil
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

// Fill fills the image with a solid color.
//
// When the image is disposed, Fill does nothing.
func (i *Image) Fill(clr color.Color) {
	i.copyCheck()
	if i.isDisposed() {
		return
	}

	var crf, cgf, cbf, caf float32
	cr, cg, cb, ca := clr.RGBA()
	crf = float32(cr) / 0xffff
	cgf = float32(cg) / 0xffff
	cbf = float32(cb) / 0xffff
	caf = float32(ca) / 0xffff
	i.image.Fill(crf, cgf, cbf, caf, i.adjustedBounds())
}

func canSkipMipmap(geom GeoM, filter builtinshader.Filter) bool {
	if filter != builtinshader.FilterLinear {
		return true
	}
	return geom.det2x2() >= 0.999
}

// DrawImageOptions represents options for DrawImage.
type DrawImageOptions struct {
	// GeoM is a geometry matrix to draw.
	// The default (zero) value is identity, which draws the image at (0, 0).
	GeoM GeoM

	// ColorScale is a scale of color.
	//
	// ColorScale is slightly different from colorm.ColorM's Scale in terms of alphas.
	// ColorScale is applied to premultiplied-alpha colors, while colorm.ColorM is applied to straight-alpha colors.
	// Thus, ColorM.Scale(r, g, b, a) equals to ColorScale.Scale(r*a, g*a, b*a, a).
	//
	// The default (zero) value is identity, which is (1, 1, 1, 1).
	ColorScale ColorScale

	// ColorM is a color matrix to draw.
	// The default (zero) value is identity, which doesn't change any color.
	//
	// Deprecated: as of v2.5. Use ColorScale or the package colorm instead.
	ColorM ColorM

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is CompositeModeCustom (Blend is used).
	//
	// Deprecated: as of v2.5. Use Blend instead.
	CompositeMode CompositeMode

	// Blend is a blending way of the source color and the destination color.
	// Blend is used only when CompositeMode is CompositeModeCustom.
	// The default (zero) value is the regular alpha blending.
	Blend Blend

	// Filter is a type of texture filter.
	// The default (zero) value is FilterNearest.
	Filter Filter
}

// adjustPosition converts the position in the *ebiten.Image coordinate to the *ui.Image coordinate.
func (i *Image) adjustPosition(x, y int) (int, int) {
	if i.isSubImage() {
		or := i.original.Bounds()
		x -= or.Min.X
		y -= or.Min.Y
		return x, y
	}

	r := i.Bounds()
	x -= r.Min.X
	y -= r.Min.Y
	return x, y
}

// adjustPositionF32 converts the position in the *ebiten.Image coordinate to the *ui.Image coordinate.
func (i *Image) adjustPositionF32(x, y float32) (float32, float32) {
	if i.isSubImage() {
		or := i.original.Bounds()
		x -= float32(or.Min.X)
		y -= float32(or.Min.Y)
		return x, y
	}

	r := i.Bounds()
	x -= float32(r.Min.X)
	y -= float32(r.Min.Y)
	return x, y
}

func (i *Image) adjustedBounds() image.Rectangle {
	b := i.Bounds()
	x, y := i.adjustPosition(b.Min.X, b.Min.Y)
	return image.Rect(x, y, x+b.Dx(), y+b.Dy())
}

func (i *Image) adjustedRegion() graphicsdriver.Region {
	b := i.Bounds()
	x, y := i.adjustPosition(b.Min.X, b.Min.Y)
	return graphicsdriver.Region{
		X:      float32(x),
		Y:      float32(y),
		Width:  float32(b.Dx()),
		Height: float32(b.Dy()),
	}
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
//   - All render targets are the same (A in A.DrawImage(B, op))
//   - All Blend values are the same
//   - All Filter values are the same
//
// A whole image and its sub-image are considered to be the same, but some
// environments like browsers might not work efficiently (#2471).
//
// Even when all the above conditions are satisfied, multiple draw commands can
// be used in really rare cases. Ebitengine images usually share an internal
// automatic texture atlas, but when you consume the atlas, or you create a huge
// image, those images cannot be on the same texture atlas. In this case, draw
// commands are separated.
// Another case is when you use an offscreen as a render source. An offscreen
// doesn't share the texture atlas with high probability.
//
// For more performance tips, see https://ebitengine.org/en/documents/performancetips.html
func (i *Image) DrawImage(img *Image, options *DrawImageOptions) {
	i.copyCheck()

	if img.isDisposed() {
		panic("ebiten: the given image to DrawImage must not be disposed")
	}
	if i.isDisposed() {
		return
	}

	if options == nil {
		options = &DrawImageOptions{}
	}

	var blend graphicsdriver.Blend
	if options.CompositeMode == CompositeModeCustom {
		blend = options.Blend.internalBlend()
	} else {
		blend = options.CompositeMode.blend().internalBlend()
	}
	filter := builtinshader.Filter(options.Filter)

	geoM := options.GeoM
	if offsetX, offsetY := i.adjustPosition(0, 0); offsetX != 0 || offsetY != 0 {
		geoM.Translate(float64(offsetX), float64(offsetY))
	}
	a, b, c, d, tx, ty := geoM.elements32()

	bounds := img.Bounds()
	sx0, sy0 := img.adjustPosition(bounds.Min.X, bounds.Min.Y)
	sx1, sy1 := img.adjustPosition(bounds.Max.X, bounds.Max.Y)
	colorm, cr, cg, cb, ca := colorMToScale(options.ColorM.affineColorM())
	cr, cg, cb, ca = options.ColorScale.apply(cr, cg, cb, ca)
	vs := i.ensureTmpVertices(4 * graphics.VertexFloatCount)
	graphics.QuadVertices(vs, float32(sx0), float32(sy0), float32(sx1), float32(sy1), a, b, c, d, tx, ty, cr, cg, cb, ca)
	is := graphics.QuadIndices()

	srcs := [graphics.ShaderImageCount]*ui.Image{img.image}

	useColorM := !colorm.IsIdentity()
	shader := builtinShader(filter, builtinshader.AddressUnsafe, useColorM)
	i.tmpUniforms = i.tmpUniforms[:0]
	if useColorM {
		var body [16]float32
		var translation [4]float32
		colorm.Elements(body[:], translation[:])
		i.tmpUniforms = shader.appendUniforms(i.tmpUniforms, map[string]any{
			builtinshader.UniformColorMBody:        body[:],
			builtinshader.UniformColorMTranslation: translation[:],
		})
	}

	i.image.DrawTriangles(srcs, vs, is, blend, i.adjustedRegion(), [graphics.ShaderImageCount]graphicsdriver.Region{img.adjustedRegion()}, shader.shader, i.tmpUniforms, false, canSkipMipmap(geoM, filter), false)
}

// Vertex represents a vertex passed to DrawTriangles.
type Vertex struct {
	// DstX and DstY represents a point on a destination image.
	DstX float32
	DstY float32

	// SrcX and SrcY represents a point on a source image.
	// Be careful that SrcX/SrcY coordinates are on the image's bounds.
	// This means that an upper-left point of a sub-image might not be (0, 0).
	SrcX float32
	SrcY float32

	// ColorR/ColorG/ColorB/ColorA represents color scaling values.
	// Their interpretation depends on the concrete draw call used:
	// - DrawTriangles: straight-alpha or premultiplied-alpha encoded color multiplier.
	//   The format is determined by ColorScaleMode in DrawTrianglesOptions.
	//   If ColorA is 0, the vertex is fully transparent and color is ignored.
	//   If ColorA is 1, the vertex has the color (ColorR, ColorG, ColorB).
	//   Vertex colors are converted to premultiplied-alpha internally and
	//   interpolated linearly respecting alpha.
	// - DrawTrianglesShader: arbitrary floating point values sent to the shader.
	//   These are interpolated linearly and independently of each other.
	ColorR float32
	ColorG float32
	ColorB float32
	ColorA float32
}

// Address represents a sampler address mode.
type Address int

const (
	// AddressUnsafe means there is no guarantee when the texture coordinates are out of range.
	AddressUnsafe Address = Address(builtinshader.AddressUnsafe)

	// AddressClampToZero means that out-of-range texture coordinates return 0 (transparent).
	AddressClampToZero Address = Address(builtinshader.AddressClampToZero)

	// AddressRepeat means that texture coordinates wrap to the other side of the texture.
	AddressRepeat Address = Address(builtinshader.AddressRepeat)
)

// FillRule is the rule whether an overlapped region is rendered with DrawTriangles(Shader).
type FillRule int

const (
	// FillAll indicates all the triangles are rendered regardless of overlaps.
	FillAll FillRule = iota

	// EvenOdd means that triangles are rendered based on the even-odd rule.
	// If and only if the number of overlaps is odd, the region is rendered.
	EvenOdd
)

// ColorScaleMode is the mode of color scales in vertices.
type ColorScaleMode int

const (
	// ColorScaleModeStraightAlpha indicates color scales in vertices are
	// straight-alpha encoded color multiplier.
	ColorScaleModeStraightAlpha ColorScaleMode = iota

	// ColorScaleModePremultipliedAlpha indicates color scales in vertices are
	// premultiplied-alpha encoded color multiplier.
	ColorScaleModePremultipliedAlpha
)

// DrawTrianglesOptions represents options for DrawTriangles.
type DrawTrianglesOptions struct {
	// ColorM is a color matrix to draw.
	// The default (zero) value is identity, which doesn't change any color.
	// ColorM is applied before vertex color scale is applied.
	//
	// Deprecated: as of v2.5. Use the package colorm instead.
	ColorM ColorM

	// ColorScaleMode is the mode of color scales in vertices.
	// ColorScaleMode affects the color calculation with vertex colors, but doesn't affect with a color matrix.
	// The default (zero) value is ColorScaleModeStraightAlpha.
	ColorScaleMode ColorScaleMode

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is CompositeModeCustom (Blend is used).
	//
	// Deprecated: as of v2.5. Use Blend instead.
	CompositeMode CompositeMode

	// Blend is a blending way of the source color and the destination color.
	// Blend is used only when CompositeMode is CompositeModeCustom.
	// The default (zero) value is the regular alpha blending.
	Blend Blend

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

	// AntiAlias indicates whether the rendering uses anti-alias or not.
	// AntiAlias is useful especially when you pass vertices from the vector package.
	//
	// AntiAlias increases internal draw calls and might affect performance.
	// Use the build tag `ebitenginedebug` to check the number of draw calls if you care.
	//
	// The default (zero) value is false.
	AntiAlias bool
}

// MaxIndicesCount is the maximum number of indices for DrawTriangles and DrawTrianglesShader.
//
// Deprecated: as of v2.6. This constant is no longer used.
const MaxIndicesCount = (1 << 16) / 3 * 3

// MaxIndicesNum is the maximum number of indices for DrawTriangles and DrawTrianglesShader.
//
// Deprecated: as of v2.4. This constant is no longer used.
const MaxIndicesNum = MaxIndicesCount

// MaxVerticesCount is the maximum number of vertices for DrawTriangles and DrawTrianglesShader.
const MaxVerticesCount = graphics.MaxVerticesCount

// DrawTriangles draws triangles with the specified vertices and their indices.
//
// img is used as a source image. img cannot be nil.
// If you want to draw triangles with a solid color, use a small white image
// and adjust the color elements in the vertices. For an actual implementation,
// see the example 'vector'.
//
// Vertex contains color values, which are interpreted as straight-alpha colors by default.
// This depends on the option's ColorScaleMode.
//
// If len(vertices) is more than MaxVerticesCount, the exceeding part is ignored.
//
// If len(indices) is not multiple of 3, DrawTriangles panics.
//
// If a value in indices is out of range of vertices, or not less than MaxVerticesCount, DrawTriangles panics.
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

	if len(vertices) > graphics.MaxVerticesCount {
		// The last part cannot be specified by indices. Just omit them.
		vertices = vertices[:graphics.MaxVerticesCount]
	}
	if len(indices)%3 != 0 {
		panic("ebiten: len(indices) % 3 must be 0")
	}
	for i, idx := range indices {
		if int(idx) >= len(vertices) {
			panic(fmt.Sprintf("ebiten: indices[%d] must be less than len(vertices) (%d) but was %d", i, len(vertices), idx))
		}
		if idx >= MaxVerticesCount {
			panic(fmt.Sprintf("ebiten: indices[%d] must be less than MaxVerticesCount %d but was %d", i, MaxVerticesCount, idx))
		}
	}

	if options == nil {
		options = &DrawTrianglesOptions{}
	}

	var blend graphicsdriver.Blend
	if options.CompositeMode == CompositeModeCustom {
		blend = options.Blend.internalBlend()
	} else {
		blend = options.CompositeMode.blend().internalBlend()
	}

	address := builtinshader.Address(options.Address)
	filter := builtinshader.Filter(options.Filter)

	colorm, cr, cg, cb, ca := colorMToScale(options.ColorM.affineColorM())

	vs := i.ensureTmpVertices(len(vertices) * graphics.VertexFloatCount)
	dst := i
	if options.ColorScaleMode == ColorScaleModeStraightAlpha {
		for i, v := range vertices {
			dx, dy := dst.adjustPositionF32(v.DstX, v.DstY)
			vs[i*graphics.VertexFloatCount] = dx
			vs[i*graphics.VertexFloatCount+1] = dy
			sx, sy := img.adjustPositionF32(v.SrcX, v.SrcY)
			vs[i*graphics.VertexFloatCount+2] = sx
			vs[i*graphics.VertexFloatCount+3] = sy
			vs[i*graphics.VertexFloatCount+4] = v.ColorR * v.ColorA * cr
			vs[i*graphics.VertexFloatCount+5] = v.ColorG * v.ColorA * cg
			vs[i*graphics.VertexFloatCount+6] = v.ColorB * v.ColorA * cb
			vs[i*graphics.VertexFloatCount+7] = v.ColorA * ca
		}
	} else {
		for i, v := range vertices {
			dx, dy := dst.adjustPositionF32(v.DstX, v.DstY)
			vs[i*graphics.VertexFloatCount] = dx
			vs[i*graphics.VertexFloatCount+1] = dy
			sx, sy := img.adjustPositionF32(v.SrcX, v.SrcY)
			vs[i*graphics.VertexFloatCount+2] = sx
			vs[i*graphics.VertexFloatCount+3] = sy
			vs[i*graphics.VertexFloatCount+4] = v.ColorR * cr
			vs[i*graphics.VertexFloatCount+5] = v.ColorG * cg
			vs[i*graphics.VertexFloatCount+6] = v.ColorB * cb
			vs[i*graphics.VertexFloatCount+7] = v.ColorA * ca
		}
	}
	is := make([]uint16, len(indices))
	copy(is, indices)

	srcs := [graphics.ShaderImageCount]*ui.Image{img.image}

	useColorM := !colorm.IsIdentity()
	shader := builtinShader(filter, address, useColorM)
	i.tmpUniforms = i.tmpUniforms[:0]
	if useColorM {
		var body [16]float32
		var translation [4]float32
		colorm.Elements(body[:], translation[:])
		i.tmpUniforms = shader.appendUniforms(i.tmpUniforms, map[string]any{
			builtinshader.UniformColorMBody:        body[:],
			builtinshader.UniformColorMTranslation: translation[:],
		})
	}

	i.image.DrawTriangles(srcs, vs, is, blend, i.adjustedRegion(), [graphics.ShaderImageCount]graphicsdriver.Region{img.adjustedRegion()}, shader.shader, i.tmpUniforms, options.FillRule == EvenOdd, filter != builtinshader.FilterLinear, options.AntiAlias)
}

// DrawTrianglesShaderOptions represents options for DrawTrianglesShader.
type DrawTrianglesShaderOptions struct {
	// CompositeMode is a composite mode to draw.
	// The default (zero) value is CompositeModeCustom (Blend is used).
	//
	// Deprecated: as of v2.5. Use Blend instead.
	CompositeMode CompositeMode

	// Blend is a blending way of the source color and the destination color.
	// Blend is used only when CompositeMode is CompositeModeCustom.
	// The default (zero) value is the regular alpha blending.
	Blend Blend

	// Uniforms is a set of uniform variables for the shader.
	// The keys are the names of the uniform variables.
	// The values must be a numeric type, or a slice or an array of a numeric type.
	// If the uniform variable type is an array, a vector or a matrix,
	// you have to specify linearly flattened values as a slice or an array.
	// For example, if the uniform variable type is [4]vec4, the length will be 16.
	//
	// If a uniform variable's name doesn't exist in Uniforms, this is treated as if zero values are specified.
	Uniforms map[string]any

	// Images is a set of the source images.
	// All the images' sizes must be the same.
	Images [4]*Image

	// FillRule indicates the rule how an overlapped region is rendered.
	//
	// The rule EvenOdd is useful when you want to render a complex polygon.
	// A complex polygon is a non-convex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
	// See examples/vector for actual usages.
	//
	// The default (zero) value is FillAll.
	FillRule FillRule

	// AntiAlias indicates whether the rendering uses anti-alias or not.
	// AntiAlias is useful especially when you pass vertices from the vector package.
	//
	// AntiAlias increases internal draw calls and might affect performance.
	// Use the build tag `ebitenginedebug` to check the number of draw calls if you care.
	//
	// The default (zero) value is false.
	AntiAlias bool
}

// Check the number of images.
var _ [len(DrawTrianglesShaderOptions{}.Images) - graphics.ShaderImageCount]struct{} = [0]struct{}{}

// DrawTrianglesShader draws triangles with the specified vertices and their indices with the specified shader.
//
// Vertex contains color values, which can be interpreted for any purpose by the shader.
//
// For the details about the shader, see https://ebitengine.org/en/documents/shader.html.
//
// If the shader unit is texels, one of the specified image is non-nil and its size is different from (width, height),
// DrawTrianglesShader panics.
// If one of the specified image is non-nil and is disposed, DrawTrianglesShader panics.
//
// If len(vertices) is more than MaxVerticesCount, the exceeding part is ignored.
//
// If len(indices) is not multiple of 3, DrawTrianglesShader panics.
//
// If a value in indices is out of range of vertices, or not less than MaxVerticesCount, DrawTrianglesShader panics.
//
// When a specified image is non-nil and is disposed, DrawTrianglesShader panics.
//
// If a specified uniform variable's length or type doesn't match with an expected one, DrawTrianglesShader panics.
//
// When the image i is disposed, DrawTrianglesShader does nothing.
func (i *Image) DrawTrianglesShader(vertices []Vertex, indices []uint16, shader *Shader, options *DrawTrianglesShaderOptions) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	if len(vertices) > graphics.MaxVerticesCount {
		// The last part cannot be specified by indices. Just omit them.
		vertices = vertices[:graphics.MaxVerticesCount]
	}
	if len(indices)%3 != 0 {
		panic("ebiten: len(indices) % 3 must be 0")
	}
	for i, idx := range indices {
		if int(idx) >= len(vertices) {
			panic(fmt.Sprintf("ebiten: indices[%d] must be less than len(vertices) (%d) but was %d", i, len(vertices), idx))
		}
		if idx >= MaxVerticesCount {
			panic(fmt.Sprintf("ebiten: indices[%d] must be less than MaxVerticesCount %d but was %d", i, MaxVerticesCount, idx))
		}
	}

	if options == nil {
		options = &DrawTrianglesShaderOptions{}
	}

	var blend graphicsdriver.Blend
	if options.CompositeMode == CompositeModeCustom {
		blend = options.Blend.internalBlend()
	} else {
		blend = options.CompositeMode.blend().internalBlend()
	}

	vs := i.ensureTmpVertices(len(vertices) * graphics.VertexFloatCount)
	dst := i
	src := options.Images[0]
	for i, v := range vertices {
		dx, dy := dst.adjustPositionF32(v.DstX, v.DstY)
		vs[i*graphics.VertexFloatCount] = dx
		vs[i*graphics.VertexFloatCount+1] = dy
		sx, sy := v.SrcX, v.SrcY
		if src != nil {
			sx, sy = src.adjustPositionF32(sx, sy)
		}
		vs[i*graphics.VertexFloatCount+2] = sx
		vs[i*graphics.VertexFloatCount+3] = sy
		vs[i*graphics.VertexFloatCount+4] = v.ColorR
		vs[i*graphics.VertexFloatCount+5] = v.ColorG
		vs[i*graphics.VertexFloatCount+6] = v.ColorB
		vs[i*graphics.VertexFloatCount+7] = v.ColorA
	}
	is := make([]uint16, len(indices))
	copy(is, indices)

	var imgs [graphics.ShaderImageCount]*ui.Image
	var imgSize image.Point
	for i, img := range options.Images {
		if img == nil {
			continue
		}
		if img.isDisposed() {
			panic("ebiten: the given image to DrawTrianglesShader must not be disposed")
		}
		if shader.unit == shaderir.Texels {
			if i == 0 {
				imgSize = img.Bounds().Size()
			} else {
				// TODO: Check imgw > 0 && imgh > 0
				if img.Bounds().Size() != imgSize {
					panic("ebiten: all the source images must be the same size with the rectangle")
				}
			}
		}
		imgs[i] = img.image
	}

	var srcRegions [graphics.ShaderImageCount]graphicsdriver.Region
	for i, img := range options.Images {
		if img == nil {
			continue
		}
		srcRegions[i] = img.adjustedRegion()
	}

	i.tmpUniforms = i.tmpUniforms[:0]
	i.tmpUniforms = shader.appendUniforms(i.tmpUniforms, options.Uniforms)

	i.image.DrawTriangles(imgs, vs, is, blend, i.adjustedRegion(), srcRegions, shader.shader, i.tmpUniforms, options.FillRule == EvenOdd, true, options.AntiAlias)
}

// DrawRectShaderOptions represents options for DrawRectShader.
type DrawRectShaderOptions struct {
	// GeoM is a geometry matrix to draw.
	// The default (zero) value is identity, which draws the rectangle at (0, 0).
	GeoM GeoM

	// ColorScale is a scale of color.
	// This scaling values are passed to the `color vec4` argument of the Fragment function in a Kage program.
	// The default (zero) value is identity, which is (1, 1, 1, 1).
	ColorScale ColorScale

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is CompositeModeCustom (Blend is used).
	//
	// Deprecated: as of v2.5. Use Blend instead.
	CompositeMode CompositeMode

	// Blend is a blending way of the source color and the destination color.
	// Blend is used only when CompositeMode is CompositeModeCustom.
	// The default (zero) value is the regular alpha blending.
	Blend Blend

	// Uniforms is a set of uniform variables for the shader.
	// The keys are the names of the uniform variables.
	// The values must be a numeric type, or a slice or an array of a numeric type.
	// If the uniform variable type is an array, a vector or a matrix,
	// you have to specify linearly flattened values as a slice or an array.
	// For example, if the uniform variable type is [4]vec4, the length will be 16.
	//
	// If a uniform variable's name doesn't exist in Uniforms, this is treated as if zero values are specified.
	Uniforms map[string]any

	// Images is a set of the source images.
	// All the images' sizes must be the same.
	Images [4]*Image
}

// Check the number of images.
var _ [len(DrawRectShaderOptions{}.Images)]struct{} = [graphics.ShaderImageCount]struct{}{}

// DrawRectShader draws a rectangle with the specified width and height with the specified shader.
//
// For the details about the shader, see https://ebitengine.org/en/documents/shader.html.
//
// When one of the specified image is non-nil and its size is different from (width, height), DrawRectShader panics.
// When one of the specified image is non-nil and is disposed, DrawRectShader panics.
//
// If a specified uniform variable's length or type doesn't match with an expected one, DrawRectShader panics.
//
// In a shader, texCoord in Fragment represents a position in a source image.
// If no source images are specified, texCoord represents the position from (0, 0) to (width, height) in pixels.
// If the unit is pixels by a compiler directive `//kage:unit pixelss`, texCoord values are valid.
// If the unit is texels (default), texCoord values still take from (0, 0) to (width, height),
// but these are invalid since texCoord is expected to be in texels in the texel-unit mode.
// This behavior is preserved for backward compatibility. It is recommended to use the pixel-unit mode to avoid confusion.
//
// If no source images are specified, imageSrc0Size returns a valid size only when the unit is pixels,
// but always returns 0 when the unit is texels (default).
//
// When the image i is disposed, DrawRectShader does nothing.
func (i *Image) DrawRectShader(width, height int, shader *Shader, options *DrawRectShaderOptions) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	if options == nil {
		options = &DrawRectShaderOptions{}
	}

	var blend graphicsdriver.Blend
	if options.CompositeMode == CompositeModeCustom {
		blend = options.Blend.internalBlend()
	} else {
		blend = options.CompositeMode.blend().internalBlend()
	}

	var imgs [graphics.ShaderImageCount]*ui.Image
	for i, img := range options.Images {
		if img == nil {
			continue
		}
		if img.isDisposed() {
			panic("ebiten: the given image to DrawRectShader must not be disposed")
		}
		if img.Bounds().Size() != image.Pt(width, height) {
			panic("ebiten: all the source images must be the same size with the rectangle")
		}
		imgs[i] = img.image
	}

	var srcRegions [graphics.ShaderImageCount]graphicsdriver.Region
	for i, img := range options.Images {
		if img == nil {
			if shader.unit == shaderir.Pixels && i == 0 {
				// Give the source size as pixels only when the unit is pixels so that users can get the source size via imageSrc0Size (#2166).
				// With the texel mode, the imageSrc0Origin and imageSrc0Size values should be in texels so the source position in pixels would not match.
				srcRegions[i] = graphicsdriver.Region{
					Width:  float32(width),
					Height: float32(height),
				}
			}
			continue
		}
		srcRegions[i] = img.adjustedRegion()
	}

	geoM := options.GeoM
	if offsetX, offsetY := i.adjustPosition(0, 0); offsetX != 0 || offsetY != 0 {
		geoM.Translate(float64(offsetX), float64(offsetY))
	}
	a, b, c, d, tx, ty := geoM.elements32()
	cr, cg, cb, ca := options.ColorScale.elements()
	vs := i.ensureTmpVertices(4 * graphics.VertexFloatCount)

	// Do not use srcRegions[0].Width and srcRegions[0].Height as these might be empty.
	graphics.QuadVertices(vs, srcRegions[0].X, srcRegions[0].Y, srcRegions[0].X+float32(width), srcRegions[0].Y+float32(height), a, b, c, d, tx, ty, cr, cg, cb, ca)
	is := graphics.QuadIndices()

	i.tmpUniforms = i.tmpUniforms[:0]
	i.tmpUniforms = shader.appendUniforms(i.tmpUniforms, options.Uniforms)

	i.image.DrawTriangles(imgs, vs, is, blend, i.adjustedRegion(), srcRegions, shader.shader, i.tmpUniforms, false, true, false)
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
//
// Successive uses of multiple various regions as rendering destination is still efficient
// when all the underlying images are the same, but some platforms like browsers might not work efficiently.
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

	var orig = i
	if i.isSubImage() {
		orig = i.original
	}

	img := &Image{
		image:    i.image,
		bounds:   r,
		original: orig,
	}
	img.addr = img

	return img
}

// Bounds returns the bounds of the image.
//
// Bounds implements the standard image.Image's Bounds.
func (i *Image) Bounds() image.Rectangle {
	if i.isDisposed() {
		panic("ebiten: the image is already disposed")
	}
	return i.bounds
}

// ColorModel returns the color model of the image.
//
// ColorModel implements the standard image.Image's ColorModel.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// ReadPixels reads the image's pixels from the image.
//
// The given pixels represent RGBA pre-multiplied alpha values.
//
// ReadPixels loads pixels from GPU to system memory if necessary, which means that ReadPixels can be slow.
//
// ReadPixels always sets a transparent color if the image is disposed.
//
// len(pixels) must be 4 * (bounds width) * (bounds height).
// If len(pixels) is not correct, ReadPixels panics.
//
// ReadPixels also works on a sub-image.
//
// Note that an important logic should not rely on values returned by ReadPixels, since
// the returned values can include very slight differences between some machines.
//
// ReadPixels can't be called outside the main loop (ebiten.Run's updating function) starts.
func (i *Image) ReadPixels(pixels []byte) {
	b := i.Bounds()
	if got, want := len(pixels), 4*b.Dx()*b.Dy(); got != want {
		panic(fmt.Sprintf("ebiten: len(pixels) must be %d but %d at ReadPixels", want, got))
	}

	if i.isDisposed() {
		for i := range pixels {
			pixels[i] = 0
		}
		return
	}

	i.image.ReadPixels(pixels, i.adjustedBounds())
}

// At returns the color of the image at (x, y).
//
// At implements the standard image.Image's At.
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
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// RGBA64At implements the standard image.RGBA64Image's RGBA64At.
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
	return color.RGBA64{R: uint16(r) * 0x101, G: uint16(g) * 0x101, B: uint16(b) * 0x101, A: uint16(a) * 0x101}
}

func (i *Image) at(x, y int) (r, g, b, a byte) {
	if i.isDisposed() {
		return 0, 0, 0, 0
	}
	if !image.Pt(x, y).In(i.Bounds()) {
		return 0, 0, 0, 0
	}

	x, y = i.adjustPosition(x, y)
	var pix [4]byte
	i.image.ReadPixels(pix[:], image.Rect(x, y, x+1, y+1))
	return pix[0], pix[1], pix[2], pix[3]
}

// Set sets the color at (x, y).
//
// Set implements the standard draw.Image's Set.
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

	dx, dy := i.adjustPosition(x, y)
	cr, cg, cb, ca := clr.RGBA()
	i.image.WritePixels([]byte{byte(cr >> 8), byte(cg >> 8), byte(cb >> 8), byte(ca >> 8)}, image.Rect(dx, dy, dx+1, dy+1))
}

// Dispose disposes the image data.
// After disposing, most of the image functions do nothing and returns meaningless values.
//
// Calling Dispose is not mandatory. GC automatically collects internal resources that no objects refer to.
// However, calling Dispose explicitly is helpful if memory usage matters.
//
// If the image is a sub-image, Dispose does nothing.
//
// When the image is disposed, Dispose does nothing.
func (i *Image) Dispose() {
	i.copyCheck()

	if i.isDisposed() {
		return
	}
	if i.isSubImage() {
		return
	}
	i.image.MarkDisposed()
	i.image = nil
}

// WritePixels replaces the pixels of the image.
//
// The given pixels are treated as RGBA pre-multiplied alpha values.
//
// len(pix) must be 4 * (bounds width) * (bounds height).
// If len(pix) is not correct, WritePixels panics.
//
// WritePixels also works on a sub-image.
//
// When the image is disposed, WritePixels does nothing.
func (i *Image) WritePixels(pixels []byte) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	// Do not need to copy pixels here.
	// * In internal/mipmap, pixels are copied when necessary.
	// * In internal/atlas, pixels are copied to make its paddings.
	i.image.WritePixels(pixels, i.adjustedBounds())
}

// ReplacePixels replaces the pixels of the image.
//
// Deprecated: as of v2.4. Use WritePixels instead.
func (i *Image) ReplacePixels(pixels []byte) {
	i.WritePixels(pixels)
}

// NewImage returns an empty image.
//
// If width or height is less than 1 or more than device-dependent maximum size, NewImage panics.
//
// NewImage should be called only when necessary.
// For example, you should avoid to call NewImage every Update or Draw call.
// Reusing the same image by Clear is much more efficient than creating a new image.
//
// NewImage panics if RunGame already finishes.
func NewImage(width, height int) *Image {
	return newImage(image.Rect(0, 0, width, height), atlas.ImageTypeRegular)
}

// NewImageOptions represents options for NewImage.
type NewImageOptions struct {
	// Unmanaged represents whether the image is unmanaged or not.
	// The default (zero) value is false, that means the image is managed.
	//
	// An unmanaged image is never on an internal automatic texture atlas.
	// A regular image is a part of an internal texture atlas, and locating them is done automatically in Ebitengine.
	// Unmanaged is useful when you want finer controls over the image for performance and memory reasons.
	Unmanaged bool
}

// NewImageWithOptions returns an empty image with the given bounds and the options.
//
// If width or height is less than 1 or more than device-dependent maximum size, NewImageWithOptions panics.
//
// The rendering origin position is (0, 0) of the given bounds.
// If DrawImage is called on a new image created by NewImageOptions,
// for example, the center of scaling and rotating is (0, 0), that might not be an upper-left position.
//
// If options is nil, the default setting is used.
//
// NewImageWithOptions should be called only when necessary.
// For example, you should avoid to call NewImageWithOptions every Update or Draw call.
// Reusing the same image by Clear is much more efficient than creating a new image.
//
// NewImageWithOptions panics if RunGame already finishes.
func NewImageWithOptions(bounds image.Rectangle, options *NewImageOptions) *Image {
	imageType := atlas.ImageTypeRegular
	if options != nil && options.Unmanaged {
		imageType = atlas.ImageTypeUnmanaged
	}
	return newImage(bounds, imageType)
}

func newImage(bounds image.Rectangle, imageType atlas.ImageType) *Image {
	if isRunGameEnded() {
		panic(fmt.Sprintf("ebiten: NewImage cannot be called after RunGame finishes"))
	}

	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 {
		panic(fmt.Sprintf("ebiten: width at NewImage must be positive but %d", width))
	}
	if height <= 0 {
		panic(fmt.Sprintf("ebiten: height at NewImage must be positive but %d", height))
	}

	i := &Image{
		image:  ui.NewImage(width, height, imageType),
		bounds: bounds,
	}
	i.addr = i
	return i
}

// NewImageFromImage creates a new image with the given image (source).
//
// If source's width or height is less than 1 or more than device-dependent maximum size, NewImageFromImage panics.
//
// NewImageFromImage should be called only when necessary.
// For example, you should avoid to call NewImageFromImage every Update or Draw call.
// Reusing the same image by Clear and WritePixels is much more efficient than creating a new image.
//
// NewImageFromImage panics if RunGame already finishes.
//
// The returned image's upper-left position is always (0, 0). The source's bounds are not respected.
func NewImageFromImage(source image.Image) *Image {
	return NewImageFromImageWithOptions(source, nil)
}

// NewImageFromImageOptions represents options for NewImageFromImage.
type NewImageFromImageOptions struct {
	// Unmanaged represents whether the image is unmanaged or not.
	// The default (zero) value is false, that means the image is managed.
	//
	// An unmanaged image is never on an internal automatic texture atlas.
	// A regular image is a part of an internal texture atlas, and locating them is done automatically in Ebitengine.
	// Unmanaged is useful when you want finer controls over the image for performance and memory reasons.
	Unmanaged bool

	// PreserveBounds represents whether the new image's bounds are the same as the given image.
	// The default (zero) value is false, that means the new image's upper-left position is adjusted to (0, 0).
	PreserveBounds bool
}

// NewImageFromImageWithOptions creates a new image with the given image (source) with the given options.
//
// If source's width or height is less than 1 or more than device-dependent maximum size, NewImageFromImageWithOptions panics.
//
// If options is nil, the default setting is used.
//
// NewImageFromImageWithOptions should be called only when necessary.
// For example, you should avoid to call NewImageFromImageWithOptions every Update or Draw call.
// Reusing the same image by Clear and WritePixels is much more efficient than creating a new image.
//
// NewImageFromImageWithOptions panics if RunGame already finishes.
func NewImageFromImageWithOptions(source image.Image, options *NewImageFromImageOptions) *Image {
	if options == nil {
		options = &NewImageFromImageOptions{}
	}

	var r image.Rectangle
	if options.PreserveBounds {
		r = source.Bounds()
	} else {
		size := source.Bounds().Size()
		r = image.Rect(0, 0, size.X, size.Y)
	}
	i := NewImageWithOptions(r, &NewImageOptions{
		Unmanaged: options.Unmanaged,
	})

	// If the given image is an Ebitengine image, use DrawImage instead of reading pixels from the source.
	// This works even before the game loop runs.
	if source, ok := source.(*Image); ok {
		op := &DrawImageOptions{}
		if options.PreserveBounds {
			b := source.Bounds()
			op.GeoM.Translate(float64(b.Min.X), float64(b.Min.Y))
		}
		i.DrawImage(source, op)
		return i
	}

	i.WritePixels(imageToBytes(source))
	return i
}

// colorMToScale returns a new color matrix and color scales that equal to the given matrix in terms of the effect.
//
// If the given matrix is merely a scaling matrix, colorMToScale returns
// an identity matrix and its scaling factors in premultiplied-alpha format.
// This is useful to optimize the rendering speed by avoiding the use of the
// color matrix and instead multiplying all vertex colors by the scale.
func colorMToScale(colorm affine.ColorM) (newColorM affine.ColorM, r, g, b, a float32) {
	if colorm.IsIdentity() {
		return colorm, 1, 1, 1, 1
	}

	if !colorm.ScaleOnly() {
		return colorm, 1, 1, 1, 1
	}

	r = colorm.At(0, 0)
	g = colorm.At(1, 1)
	b = colorm.At(2, 2)
	a = colorm.At(3, 3)

	// Color matrices work on non-premultiplied colors.
	// This color matrix can only make colors darker or equal,
	// and thus can never invoke color clamping.
	// Thus the simpler vertex color scale based shader can be used.
	//
	// Negative color values can become positive and out-of-range
	// after applying to vertex colors below, which can make the min() in the shader kick in.
	//
	// Alpha values smaller than 0, combined with negative vertex colors,
	// can also make the min() kick in, so that shall be ruled out too.
	if r < 0 || g < 0 || b < 0 || a < 0 || r > 1 || g > 1 || b > 1 {
		return colorm, 1, 1, 1, 1
	}

	return affine.ColorMIdentity{}, r * a, g * a, b * a, a
}

func (i *Image) ensureTmpVertices(n int) []float32 {
	if cap(i.tmpVertices) < n {
		i.tmpVertices = make([]float32, n)
	}
	return i.tmpVertices[:n]
}

// private implements FinalScreen.
func (*Image) private() {
}
