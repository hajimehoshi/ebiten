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
	"math"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
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

	// tmpIndices must not be reused until ui.Image.Draw* is called.
	tmpIndices []uint32

	// tmpUniforms must not be reused until ui.Image.Draw* is called.
	tmpUniforms []uint32

	// subImageCache is a cache for sub-images.
	// subImageCache is valid only when the image is not a sub-image.
	subImageCache map[image.Rectangle]*Image

	// subImageGCLastTick is the last tick when old sub images are removed from the cache.
	subImageGCLastTick int64

	// subImageCacheM is a mutex for subImageCache.
	// subImageCache can be accessed from the image and its sub-images at the same time,
	// so the map must be protected by a mutex.
	subImageCacheM sync.Mutex

	// atime is the last access time.
	// atime needs to be an atomic value since a sub-image atime can be accessed from its original image.
	atime atomic.Int64

	// usageCallbacks are callbacks that are invoked when the image is used.
	// usageCallbacks is valid only when the image is not a sub-image.
	usageCallbacks map[int64]func()

	// inUsageCallbacks reports whether the image is in usageCallbacks.
	inUsageCallbacks bool

	// Do not add a 'buffering' member that are resolved lazily.
	// This tends to forget resolving the buffer easily (#2362).
}

func (i *Image) copyCheck() {
	if i.addr != i {
		panic("ebiten: illegal use of non-zero Image copied by value")
	}
}

func (i *Image) updateAccessTime() {
	i.atime.Store(Tick())
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

	i.invokeUsageCallbacks()

	i.updateAccessTime()

	var crf, cgf, cbf, caf float32
	cr, cg, cb, ca := clr.RGBA()
	crf = float32(cr) / 0xffff
	cgf = float32(cg) / 0xffff
	cbf = float32(cb) / 0xffff
	caf = float32(ca) / 0xffff
	i.image.Fill(crf, cgf, cbf, caf, i.adjustedBounds())
}

func canSkipMipmap(det float32, filter builtinshader.Filter) bool {
	if filter != builtinshader.FilterLinear {
		return true
	}
	return math.Abs(float64(det)) >= 0.999
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

	// DisableMipmaps disables mipmaps.
	// When Filter is FilterLinear and GeoM shrinks the image, mipmaps are used by default.
	// Mipmap is useful to render a shrunk image with high quality.
	// However, mipmaps can be expensive, especially on mobiles.
	// When DisableMipmaps is true, mipmap is not used.
	// When Filter is not FilterLinear, DisableMipmaps is ignored.
	//
	// The default (zero) value is false.
	DisableMipmaps bool
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

	i.invokeUsageCallbacks()
	img.invokeUsageCallbacks()

	i.updateAccessTime()
	img.updateAccessTime()

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
	det := a*d - b*c
	if det == 0 {
		return
	}

	bounds := img.Bounds()
	sx0, sy0 := img.adjustPosition(bounds.Min.X, bounds.Min.Y)
	sx1, sy1 := img.adjustPosition(bounds.Max.X, bounds.Max.Y)
	colorm, cr, cg, cb, ca := colorMToScale(options.ColorM.affineColorM())
	cr, cg, cb, ca = options.ColorScale.apply(cr, cg, cb, ca)
	vs := i.ensureTmpVertices(4 * graphics.VertexFloatCount)
	graphics.QuadVerticesFromSrcAndMatrix(vs, float32(sx0), float32(sy0), float32(sx1), float32(sy1), a, b, c, d, tx, ty, cr, cg, cb, ca)
	is := graphics.QuadIndices()

	srcs := [graphics.ShaderSrcImageCount]*ui.Image{img.image}

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

	dr := i.adjustedBounds()
	hint := restorable.HintNone
	if overwritesDstRegion(options.Blend, dr, geoM, sx0, sy0, sx1, sy1) {
		hint = restorable.HintOverwriteDstRegion
	}

	skipMipmap := options.DisableMipmaps
	if !skipMipmap {
		skipMipmap = canSkipMipmap(det, filter)
	}
	i.image.DrawTriangles(srcs, vs, is, blend, dr, [graphics.ShaderSrcImageCount]image.Rectangle{img.adjustedBounds()}, shader.shader, i.tmpUniforms, graphicsdriver.FillRuleFillAll, skipMipmap, false, hint)
}

// overwritesDstRegion reports whether the given parameters overwrite the destination region completely.
func overwritesDstRegion(blend Blend, dstRegion image.Rectangle, geoM GeoM, sx0, sy0, sx1, sy1 int) bool {
	// TODO: More precisely, BlendFactorDestinationRGB, BlendFactorDestinationAlpha, and operations should be checked.
	if blend != BlendCopy && blend != BlendClear {
		return false
	}
	// Check the result vertices is not a rotated rectangle.
	if geoM.b != 0 || geoM.c != 0 {
		return false
	}
	// Check the result vertices completely covers dstRegion.
	x0, y0 := geoM.Apply(float64(sx0), float64(sy0))
	x1, y1 := geoM.Apply(float64(sx1), float64(sy1))
	if float64(dstRegion.Min.X) < x0 || float64(dstRegion.Min.Y) < y0 || float64(dstRegion.Max.X) > x1 || float64(dstRegion.Max.Y) > y1 {
		return false
	}
	return true
}

// Vertex represents a vertex passed to DrawTriangles.
type Vertex struct {
	// DstX and DstY represents a point on a destination image.
	DstX float32
	DstY float32

	// SrcX and SrcY represents a point on a source image.
	// Be careful that SrcX/SrcY coordinates are on the image's bounds.
	// This means that an upper-left point of a sub-image might not be (0, 0).
	//
	// Before passing vertices to a Kage shader, SrcX/SrcY are converted to texture coordinates of the first image,
	// which is DrawRectShaderOptions.Image[0] or DrawTrianglesShaderOptions.Images[0].
	// If the image is nil, SrcX/SrcY are not converted and used as-is.
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

	// Custom0/Custom1/Custom2/Custom3 represents general-purpose values passed to the shader.
	// In order to use them, Fragment must have an additional vec4 argument.
	//
	// These values are valid only when DrawTrianglesShader is used.
	// In other cases, these values are ignored.
	Custom0 float32
	Custom1 float32
	Custom2 float32
	Custom3 float32
}

var _ [0]byte = [unsafe.Sizeof(Vertex{}) - unsafe.Sizeof(float32(0))*graphics.VertexFloatCount]byte{}

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
//
// Deprecated: as of v2.9.
type FillRule int

const (
	// FillRuleFillAll indicates all the triangles are rendered regardless of overlaps.
	//
	// Deprecated: as of v2.9.
	FillRuleFillAll FillRule = FillRule(graphicsdriver.FillRuleFillAll)

	// FillRuleNonZero means that triangles are rendered based on the non-zero rule.
	// If and only if the number of overlaps is not 0, the region is rendered.
	//
	// Deprecated: as of v2.9.
	FillRuleNonZero FillRule = FillRule(graphicsdriver.FillRuleNonZero)

	// FillRuleEvenOdd means that triangles are rendered based on the even-odd rule.
	// If and only if the number of overlaps is odd, the region is rendered.
	//
	// Deprecated: as of v2.9.
	FillRuleEvenOdd FillRule = FillRule(graphicsdriver.FillRuleEvenOdd)
)

const (
	// FillAll indicates all the triangles are rendered regardless of overlaps.
	//
	// Deprecated: as of v2.8. Use FillRuleFillAll instead.
	FillAll = FillRuleFillAll

	// NonZero means that triangles are rendered based on the non-zero rule.
	// If and only if the number of overlaps is not 0, the region is rendered.
	//
	// Deprecated: as of v2.8. Use FillRuleNonZero instead.
	NonZero = FillRuleNonZero

	// EvenOdd means that triangles are rendered based on the even-odd rule.
	// If and only if the number of overlaps is odd, the region is rendered.
	//
	// Deprecated: as of v2.8. Use FillRuleEvenOdd instead.
	EvenOdd = FillRuleEvenOdd
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
	// The rules FillRuleNonZero and FillRuleEvenOdd are useful when you want to render a complex polygon.
	// A complex polygon is a non-convex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
	// See examples/vector for actual usages.
	//
	// The default (zero) value is FillRuleFillAll.
	//
	// Deprecated: as of v2.9. Use [github.com/hajimehoshi/ebiten/v2/vector.FillPath] instead.
	FillRule FillRule

	// AntiAlias indicates whether the rendering uses anti-alias or not.
	// AntiAlias is useful especially when you pass vertices from the vector package.
	//
	// AntiAlias increases internal draw calls and might affect performance.
	// Use the build tag `ebitenginedebug` to check the number of draw calls if you care.
	//
	// The default (zero) value is false.//
	//
	// Deprecated: as of v2.9. Use [github.com/hajimehoshi/ebiten/v2/vector.FillPath] instead.
	AntiAlias bool

	// DisableMipmaps disables mipmaps.
	// When Filter is FilterLinear and GeoM shrinks the image, mipmaps are used by default.
	// Mipmap is useful to render a shrunk image with high quality.
	// However, mipmaps can be expensive, especially on mobiles.
	// When DisableMipmaps is true, mipmap is not used.
	// When Filter is not FilterLinear, DisableMipmaps is ignored.
	//
	// The default (zero) value is false.
	DisableMipmaps bool
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
//
// Deprecated: as of v2.7. Use MaxVertexCount instead.
const MaxVerticesCount = graphicscommand.MaxVertexCount

// MaxVertexCount is the maximum number of vertices for DrawTriangles and DrawTrianglesShader.
const MaxVertexCount = graphicscommand.MaxVertexCount

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
// If len(vertices) is more than MaxVertexCount, the exceeding part is ignored.
//
// If len(indices) is not multiple of 3, DrawTriangles panics.
//
// If a value in indices is out of range of vertices, or not less than MaxVertexCount, DrawTriangles panics.
//
// The rule in which DrawTriangles works effectively is same as DrawImage's.
//
// When the given image is disposed, DrawTriangles panics.
//
// When the image i is disposed, DrawTriangles does nothing.
func (i *Image) DrawTriangles(vertices []Vertex, indices []uint16, img *Image, options *DrawTrianglesOptions) {
	is := i.ensureTmpIndices(len(indices))
	for i := range is {
		is[i] = uint32(indices[i])
	}
	i.DrawTriangles32(vertices, is, img, options)
}

// DrawTriangles32 draws triangles with the specified vertices and their indices.
// DrawTriangles32 is the version of DrawTriangles with uint32 indices.
//
// img is used as a source image. img cannot be nil.
// If you want to draw triangles with a solid color, use a small white image
// and adjust the color elements in the vertices. For an actual implementation,
// see the example 'vector'.
//
// Vertex contains color values, which are interpreted as straight-alpha colors by default.
// This depends on the option's ColorScaleMode.
//
// If len(vertices) is more than MaxVertexCount, the exceeding part is ignored.
//
// If len(indices) is not multiple of 3, DrawTriangles32 panics.
//
// If a value in indices is out of range of vertices, or not less than MaxVertexCount, DrawTriangles32 panics.
//
// The rule in which DrawTriangles32 works effectively is same as DrawImage's.
//
// When the given image is disposed, DrawTriangles32 panics.
//
// When the image i is disposed, DrawTriangles32 does nothing.
func (i *Image) DrawTriangles32(vertices []Vertex, indices []uint32, img *Image, options *DrawTrianglesOptions) {
	i.copyCheck()

	if img != nil && img.isDisposed() {
		panic("ebiten: the given image to DrawTriangles must not be disposed")
	}
	if i.isDisposed() {
		return
	}

	if len(indices) == 0 {
		return
	}

	i.invokeUsageCallbacks()
	img.invokeUsageCallbacks()

	img.updateAccessTime()
	i.updateAccessTime()

	if len(vertices) > graphicscommand.MaxVertexCount {
		// The last part cannot be specified by indices. Just omit them.
		vertices = vertices[:graphicscommand.MaxVertexCount]
	}
	if len(indices)%3 != 0 {
		panic("ebiten: len(indices) % 3 must be 0")
	}
	for i, idx := range indices {
		if int(idx) >= len(vertices) {
			panic(fmt.Sprintf("ebiten: indices[%d] must be less than len(vertices) (%d) but was %d", i, len(vertices), idx))
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
		// Avoid using `for i, v := range vertices` as adding `v` creates a copy from `vertices` unnecessarily on each loop (#3103).
		for i := range vertices {
			// Create a temporary slice to reduce boundary checks.
			vs := vs[i*graphics.VertexFloatCount : i*graphics.VertexFloatCount+8]
			dx, dy := dst.adjustPositionF32(vertices[i].DstX, vertices[i].DstY)
			vs[0] = dx
			vs[1] = dy
			sx, sy := img.adjustPositionF32(vertices[i].SrcX, vertices[i].SrcY)
			vs[2] = sx
			vs[3] = sy
			vs[4] = vertices[i].ColorR * vertices[i].ColorA * cr
			vs[5] = vertices[i].ColorG * vertices[i].ColorA * cg
			vs[6] = vertices[i].ColorB * vertices[i].ColorA * cb
			vs[7] = vertices[i].ColorA * ca
		}
	} else {
		// See comment above (#3103).
		for i := range vertices {
			// Create a temporary slice to reduce boundary checks.
			vs := vs[i*graphics.VertexFloatCount : i*graphics.VertexFloatCount+8]
			dx, dy := dst.adjustPositionF32(vertices[i].DstX, vertices[i].DstY)
			vs[0] = dx
			vs[1] = dy
			sx, sy := img.adjustPositionF32(vertices[i].SrcX, vertices[i].SrcY)
			vs[2] = sx
			vs[3] = sy
			vs[4] = vertices[i].ColorR * cr
			vs[5] = vertices[i].ColorG * cg
			vs[6] = vertices[i].ColorB * cb
			vs[7] = vertices[i].ColorA * ca
		}
	}

	srcs := [graphics.ShaderSrcImageCount]*ui.Image{img.image}

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

	skipMipmap := options.DisableMipmaps
	if !skipMipmap {
		skipMipmap = filter != builtinshader.FilterLinear
	}
	i.image.DrawTriangles(srcs, vs, indices, blend, i.adjustedBounds(), [graphics.ShaderSrcImageCount]image.Rectangle{img.adjustedBounds()}, shader.shader, i.tmpUniforms, graphicsdriver.FillRule(options.FillRule), skipMipmap, options.AntiAlias, restorable.HintNone)
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
	// The values must be a numeric/boolean type, or a slice or an array of a numeric/boolean type.
	// If the uniform variable type is an array, a vector or a matrix,
	// you have to specify linearly flattened values as a slice or an array.
	// For example, if the uniform variable type is [4]vec4, the length will be 16.
	//
	// If a uniform variable's name doesn't exist in Uniforms, this is treated as if zero values are specified.
	Uniforms map[string]any

	// Images is a set of the source images.
	// In the texel mode, all the image sizes must be the same.
	// The pixel mode allows images of different sizes.
	Images [4]*Image

	// FillRule indicates the rule how an overlapped region is rendered.
	//
	// The rules FillRuleNonZero and FillRuleEvenOdd are useful when you want to render a complex polygon.
	// A complex polygon is a non-convex polygon like a concave polygon, a polygon with holes, or a self-intersecting polygon.
	// See examples/vector for actual usages.
	//
	// The default (zero) value is FillRuleFillAll.
	//
	// Deprecated: as of v2.9. Use [github.com/hajimehoshi/ebiten/v2/vector.FillPath] instead.
	FillRule FillRule

	// AntiAlias indicates whether the rendering uses anti-alias or not.
	// AntiAlias is useful especially when you pass vertices from the vector package.
	//
	// AntiAlias increases internal draw calls and might affect performance.
	// Use the build tag `ebitenginedebug` to check the number of draw calls if you care.
	//
	// The default (zero) value is false.
	//
	// Deprecated: as of v2.9. Use [github.com/hajimehoshi/ebiten/v2/vector.FillPath] instead.
	AntiAlias bool
}

// Check the number of images.
var _ [len(DrawTrianglesShaderOptions{}.Images) - graphics.ShaderSrcImageCount]struct{} = [0]struct{}{}

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
// If len(vertices) is more than MaxVertexCount, the exceeding part is ignored.
//
// If len(indices) is not multiple of 3, DrawTrianglesShader panics.
//
// If a value in indices is out of range of vertices, or not less than MaxVertexCount, DrawTrianglesShader panics.
//
// When a specified image is non-nil and is disposed, DrawTrianglesShader panics.
//
// If a specified uniform variable's length or type doesn't match with an expected one, DrawTrianglesShader panics.
//
// Even if a result is an invalid color as a premultiplied-alpha color, i.e. an alpha value exceeds other color values,
// the value is kept and is not clamped.
//
// When the image i is disposed, DrawTrianglesShader does nothing.
func (i *Image) DrawTrianglesShader(vertices []Vertex, indices []uint16, shader *Shader, options *DrawTrianglesShaderOptions) {
	is := i.ensureTmpIndices(len(indices))
	for i := range is {
		is[i] = uint32(indices[i])
	}
	i.DrawTrianglesShader32(vertices, is, shader, options)
}

// DrawTrianglesShader32 draws triangles with the specified vertices and their indices with the specified shader.
// DrawTrianglesShader32 is the version of DrawTrianglesShader with uint32 indices.
//
// Vertex contains color values, which can be interpreted for any purpose by the shader.
//
// For the details about the shader, see https://ebitengine.org/en/documents/shader.html.
//
// If the shader unit is texels, one of the specified image is non-nil and its size is different from (width, height),
// DrawTrianglesShader32 panics.
// If one of the specified image is non-nil and is disposed, DrawTrianglesShader32 panics.
//
// If len(vertices) is more than MaxVertexCount, the exceeding part is ignored.
//
// If len(indices) is not multiple of 3, DrawTrianglesShader32 panics.
//
// If a value in indices is out of range of vertices, or not less than MaxVertexCount, DrawTrianglesShader32 panics.
//
// When a specified image is non-nil and is disposed, DrawTrianglesShader32 panics.
//
// If a specified uniform variable's length or type doesn't match with an expected one, DrawTrianglesShader32 panics.
//
// Even if a result is an invalid color as a premultiplied-alpha color, i.e. an alpha value exceeds other color values,
// the value is kept and is not clamped.
//
// When the image i is disposed, DrawTrianglesShader32 does nothing.
func (i *Image) DrawTrianglesShader32(vertices []Vertex, indices []uint32, shader *Shader, options *DrawTrianglesShaderOptions) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	if shader.isDisposed() {
		panic("ebiten: the given shader to DrawTrianglesShader must not be disposed")
	}

	if len(indices) == 0 {
		return
	}

	i.invokeUsageCallbacks()
	if options != nil {
		for _, img := range options.Images {
			if img == nil {
				continue
			}
			img.invokeUsageCallbacks()
		}
	}

	if options != nil {
		for _, img := range options.Images {
			if img == nil {
				continue
			}
			img.updateAccessTime()
		}
	}
	i.updateAccessTime()

	if len(vertices) > graphicscommand.MaxVertexCount {
		// The last part cannot be specified by indices. Just omit them.
		vertices = vertices[:graphicscommand.MaxVertexCount]
	}
	if len(indices)%3 != 0 {
		panic("ebiten: len(indices) % 3 must be 0")
	}
	for i, idx := range indices {
		if int(idx) >= len(vertices) {
			panic(fmt.Sprintf("ebiten: indices[%d] must be less than len(vertices) (%d) but was %d", i, len(vertices), idx))
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
	// Avoid using `for i, v := range vertices` as adding `v` creates a copy from `vertices` unnecessarily on each loop (#3103).
	for i := range vertices {
		// Create a temporary slice to reduce boundary checks.
		vs := vs[i*graphics.VertexFloatCount : i*graphics.VertexFloatCount+12]
		dx, dy := dst.adjustPositionF32(vertices[i].DstX, vertices[i].DstY)
		vs[0] = dx
		vs[1] = dy
		sx, sy := vertices[i].SrcX, vertices[i].SrcY
		if src != nil {
			sx, sy = src.adjustPositionF32(sx, sy)
		}
		vs[2] = sx
		vs[3] = sy
		vs[4] = vertices[i].ColorR
		vs[5] = vertices[i].ColorG
		vs[6] = vertices[i].ColorB
		vs[7] = vertices[i].ColorA
		vs[8] = vertices[i].Custom0
		vs[9] = vertices[i].Custom1
		vs[10] = vertices[i].Custom2
		vs[11] = vertices[i].Custom3
	}

	var imgs [graphics.ShaderSrcImageCount]*ui.Image
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

	var srcRegions [graphics.ShaderSrcImageCount]image.Rectangle
	for i, img := range options.Images {
		if img == nil {
			continue
		}
		srcRegions[i] = img.adjustedBounds()
	}

	i.tmpUniforms = i.tmpUniforms[:0]
	i.tmpUniforms = shader.appendUniforms(i.tmpUniforms, options.Uniforms)

	i.image.DrawTriangles(imgs, vs, indices, blend, i.adjustedBounds(), srcRegions, shader.shader, i.tmpUniforms, graphicsdriver.FillRule(options.FillRule), true, options.AntiAlias, restorable.HintNone)
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
	// The values must be a numeric/boolean type, or a slice or an array of a numeric/boolean type.
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
var _ [len(DrawRectShaderOptions{}.Images)]struct{} = [graphics.ShaderSrcImageCount]struct{}{}

// DrawRectShader draws a rectangle with the specified width and height with the specified shader.
//
// For the details about the shader, see https://ebitengine.org/en/documents/shader.html.
//
// When one of the specified image is non-nil and its size is different from (width, height), DrawRectShader panics.
// When one of the specified image is non-nil and is disposed, DrawRectShader panics.
//
// If a specified uniform variable's length or type doesn't match with an expected one, DrawRectShader panics.
//
// In a shader, srcPos in Fragment represents a position in a source image.
// If no source images are specified, srcPos represents the position from (0, 0) to (width, height) in pixels.
// If the unit is pixels by a compiler directive `//kage:unit pixelss`, srcPos values are valid.
// If the unit is texels (default), srcPos values still take from (0, 0) to (width, height),
// but these are invalid since srcPos is expected to be in texels in the texel-unit mode.
// This behavior is preserved for backward compatibility. It is recommended to use the pixel-unit mode to avoid confusion.
//
// If no source images are specified, imageSrc0Size returns a valid size only when the unit is pixels,
// but always returns 0 when the unit is texels (default).
//
// Even if a result is an invalid color as a premultiplied-alpha color, i.e. an alpha value exceeds other color values,
// the value is kept and is not clamped.
//
// When the image i is disposed, DrawRectShader does nothing.
func (i *Image) DrawRectShader(width, height int, shader *Shader, options *DrawRectShaderOptions) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	if shader.isDisposed() {
		panic("ebiten: the given shader to DrawRectShader must not be disposed")
	}

	if options != nil {
		for _, img := range options.Images {
			if img == nil {
				continue
			}
			img.invokeUsageCallbacks()
		}
	}
	i.invokeUsageCallbacks()

	if options != nil {
		for _, img := range options.Images {
			if img == nil {
				continue
			}
			img.updateAccessTime()
		}
	}
	i.updateAccessTime()

	if options == nil {
		options = &DrawRectShaderOptions{}
	}

	var blend graphicsdriver.Blend
	if options.CompositeMode == CompositeModeCustom {
		blend = options.Blend.internalBlend()
	} else {
		blend = options.CompositeMode.blend().internalBlend()
	}

	var imgs [graphics.ShaderSrcImageCount]*ui.Image
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

	var srcRegions [graphics.ShaderSrcImageCount]image.Rectangle
	for i, img := range options.Images {
		if img == nil {
			if shader.unit == shaderir.Pixels && i == 0 {
				// Give the source size as pixels only when the unit is pixels so that users can get the source size via imageSrc0Size (#2166).
				// With the texel mode, the imageSrc0Origin and imageSrc0Size values should be in texels so the source position in pixels would not match.
				srcRegions[i] = image.Rect(0, 0, width, height)
			}
			continue
		}
		srcRegions[i] = img.adjustedBounds()
	}

	geoM := options.GeoM
	if offsetX, offsetY := i.adjustPosition(0, 0); offsetX != 0 || offsetY != 0 {
		geoM.Translate(float64(offsetX), float64(offsetY))
	}
	a, b, c, d, tx, ty := geoM.elements32()
	if det := a*d - b*c; det == 0 {
		return
	}
	cr, cg, cb, ca := options.ColorScale.elements()
	vs := i.ensureTmpVertices(4 * graphics.VertexFloatCount)

	// Do not use srcRegions[0].Dx() and srcRegions[0].Dy() as these might be empty.
	graphics.QuadVerticesFromSrcAndMatrix(vs,
		float32(srcRegions[0].Min.X), float32(srcRegions[0].Min.Y),
		float32(srcRegions[0].Min.X+width), float32(srcRegions[0].Min.Y+height),
		a, b, c, d, tx, ty, cr, cg, cb, ca)
	is := graphics.QuadIndices()

	i.tmpUniforms = i.tmpUniforms[:0]
	i.tmpUniforms = shader.appendUniforms(i.tmpUniforms, options.Uniforms)

	dr := i.adjustedBounds()
	hint := restorable.HintNone
	// Do not use srcRegions[0].Dx() and srcRegions[0].Dy() as these might be empty.
	if overwritesDstRegion(options.Blend, dr, geoM, srcRegions[0].Min.X, srcRegions[0].Min.Y, srcRegions[0].Min.X+width, srcRegions[0].Min.Y+height) {
		hint = restorable.HintOverwriteDstRegion
	}

	i.image.DrawTriangles(imgs, vs, is, blend, dr, srcRegions, shader.shader, i.tmpUniforms, graphicsdriver.FillRuleFillAll, true, false, hint)
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

	if i.isSubImage() {
		return i.original.SubImage(r.Intersect(i.Bounds()))
	}

	r = r.Intersect(i.Bounds())
	// Need to check Empty explicitly. See the standard image package implementations.
	if r.Empty() {
		r = image.Rectangle{}
	}

	i.subImageCacheM.Lock()
	defer i.subImageCacheM.Unlock()

	// The image might already be disposed in another goroutine.
	// Recheck this.
	if i.isDisposed() {
		return nil
	}

	if img, ok := i.subImageCache[r]; ok {
		img.updateAccessTime()
		return img
	}

	if tick := Tick(); i.subImageGCLastTick < tick {
		i.subImageGCLastTick = tick

		for _, img := range i.subImageCache {
			if img.atime.Load()+60 < tick {
				delete(i.subImageCache, img.bounds)
			}
		}
	}

	img := &Image{
		image:    i.image,
		bounds:   r,
		original: i,
	}
	img.addr = img

	if i.subImageCache == nil {
		i.subImageCache = map[image.Rectangle]*Image{}
	}
	i.subImageCache[r] = img
	img.updateAccessTime()

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

	i.invokeUsageCallbacks()

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

	i.invokeUsageCallbacks()

	x, y = i.adjustPosition(x, y)
	var pix [4]byte
	i.image.ReadPixels(pix[:], image.Rect(x, y, x+1, y+1))
	return pix[0], pix[1], pix[2], pix[3]
}

// Set sets the color at (x, y).
//
// Set implements the standard draw.Image's Set.
//
// If (x, y) is outside the image bounds, Set does nothing.
//
// Even if a result is an invalid color as a premultiplied-alpha color, i.e. an alpha value exceeds other color values,
// the value is kept and is not clamped.
//
// If the image is disposed, Set does nothing.
//
// For performance, it is recommended to use WritePixels instead of Set whenever possible.
func (i *Image) Set(x, y int, clr color.Color) {
	i.copyCheck()
	if i.isDisposed() {
		return
	}

	i.invokeUsageCallbacks()

	i.updateAccessTime()

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
// If the image is disposed, Dispose does nothing.
//
// Deprecated: as of v2.7. Use Deallocate instead.
func (i *Image) Dispose() {
	i.copyCheck()

	if i.isDisposed() {
		return
	}
	if i.isSubImage() {
		return
	}
	i.image.Deallocate()
	i.image = nil
	i.subImageCacheM.Lock()
	i.subImageCache = nil
	i.subImageCacheM.Unlock()
	i.usageCallbacks = nil
}

// Deallocate clears the image and deallocates the internal state of the image.
// Even after Deallocate is called, the image is still available.
// In this case, the image's internal state is allocated again.
//
// Usually, you don't have to call Deallocate since the internal state is automatically released by GC.
// However, if you are sure that the image is no longer used but not sure how this image object is referred,
// you can call Deallocate to make sure that the internal state is deallocated.
//
// If the image is a sub-image, Deallocate does nothing.
//
// If the image is disposed, Deallocate does nothing.
func (i *Image) Deallocate() {
	i.copyCheck()

	if i.isDisposed() {
		return
	}
	if i.isSubImage() {
		return
	}
	i.image.Deallocate()
	i.usageCallbacks = nil
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
// Even if a result is an invalid color as a premultiplied-alpha color, i.e. an alpha value exceeds other color values,
// the value is kept and is not clamped.
//
// When the image is disposed, WritePixels does nothing.
func (i *Image) WritePixels(pixels []byte) {
	i.copyCheck()

	if i.isDisposed() {
		return
	}

	i.invokeUsageCallbacks()

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
		panic("ebiten: NewImage cannot be called after RunGame finishes")
	}

	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 {
		panic(fmt.Sprintf("ebiten: width at NewImage must be positive but %d", width))
	}
	if height <= 0 {
		panic(fmt.Sprintf("ebiten: height at NewImage must be positive but %d", height))
	}

	i := &Image{
		image:  ui.Get().NewImage(width, height, imageType),
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

func (i *Image) ensureTmpIndices(n int) []uint32 {
	if cap(i.tmpIndices) < n {
		i.tmpIndices = make([]uint32, n)
	}
	return i.tmpIndices[:n]
}

// private implements FinalScreen.
func (*Image) private() {
}

// Do not use usage callbacks except for Ebitengine packages.
// There is no guarantee for compatibility of this function.

var currentCallbackToken atomic.Int64

//go:linkname addUsageCallback
func addUsageCallback(img *Image, callback func()) int64 {
	return img.addUsageCallback(callback)
}

func (i *Image) addUsageCallback(callback func()) int64 {
	if i.isSubImage() {
		return i.original.addUsageCallback(callback)
	}
	if i.usageCallbacks == nil {
		i.usageCallbacks = map[int64]func(){}
	}
	token := currentCallbackToken.Add(1)
	i.usageCallbacks[token] = callback
	return token
}

//go:linkname removeUsageCallback
func removeUsageCallback(img *Image, token int64) {
	img.removeUsageCallback(token)
}

func (i *Image) removeUsageCallback(token int64) {
	if i.isSubImage() {
		i.original.removeUsageCallback(token)
		return
	}
	delete(i.usageCallbacks, token)
}

func (i *Image) invokeUsageCallbacks() {
	if i.isSubImage() {
		i.original.invokeUsageCallbacks()
		return
	}

	if i.inUsageCallbacks {
		return
	}

	i.inUsageCallbacks = true
	defer func() {
		i.inUsageCallbacks = false
	}()

	for _, cb := range i.usageCallbacks {
		cb()
	}
}
