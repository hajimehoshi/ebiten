# Kage recipes

Companion to [../SKILL.md](../SKILL.md). Every snippet assumes
`//kage:unit pixels` and is written for atlas-relative coordinates.

## Normalization

```go
// Destination position, normalized to 0..1 over the destination image.
func dstUV(dstPos vec4) vec2 {
	return (dstPos.xy - imageDstOrigin()) / imageDstSize()
}

// Source position, normalized to 0..1 over source image 0.
func srcUV(srcPos vec2) vec2 {
	return (srcPos - imageSrc0Origin()) / imageSrc0Size()
}

// The inverse: a 0..1 coordinate back to a samplable position on source 0.
func srcPosOf(uv vec2) vec2 {
	return uv*imageSrc0Size() + imageSrc0Origin()
}
```

The two source helpers need a non-empty source region. `imageSrc0Size()` is zero
when `DrawTrianglesShader` runs with `Images[0] == nil`, so pass the size as a
uniform in that case. `DrawRectShader` synthesizes source region 0 from the
drawn rectangle in the pixel unit, so it is safe there with or without an image.

`dstUV` has no such precondition, but note that it normalizes over the whole
destination image, not over the area being drawn.

## Transforming a coordinate

Subtract the origin, transform, add it back. Rotation about the image centre:

```go
func rotateAroundCenter(srcPos vec2, angle float) vec2 {
	p := srcPos - imageSrc0Origin() - imageSrc0Size()/2
	s, c := sin(angle), cos(angle)
	p = vec2(p.x*c-p.y*s, p.x*s+p.y*c)
	return p + imageSrc0Size()/2 + imageSrc0Origin()
}
```

## Sampling another source image

Every slot shares image 0's coordinate space, so pass `srcPos` through
unchanged, whatever the images' sizes.

```go
func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return imageSrc0At(srcPos) * imageSrc1At(srcPos).a
}
```

To stretch image 1 across image 0 instead of aligning it pixel for pixel,
normalize in image 0's space, scale by image 1's size, and add **image 0's**
origin.

```go
func src1PosOf(srcPos vec2) vec2 {
	uv := (srcPos - imageSrc0Origin()) / imageSrc0Size()
	return uv*imageSrc1Size() + imageSrc0Origin()
}
```

## Addressing modes

`imageSrcNAt` returns `vec4(0)` outside the image. For the other two behaviours:

```go
// Clamp to edge.
func clampToEdge(pos vec2) vec2 {
	const epsilon = 0.001
	origin := imageSrc0Origin()
	return clamp(pos, origin, origin+imageSrc0Size()-vec2(epsilon))
}

// Repeat.
func repeat(pos vec2) vec2 {
	origin := imageSrc0Origin()
	return mod(pos-origin, imageSrc0Size()) + origin
}
```

Both produce a position that is provably inside the region, which is the
precondition `imageSrc0UnsafeAt` needs: it skips the bounds check, so outside
the region its result is undefined.

## Bilinear sampling

Sampling is nearest-neighbour; interpolate by hand when the draw scales up.

```go
func bilinearAt(pos vec2) vec4 {
	p := pos - vec2(0.5)
	f := fract(p)
	base := floor(p) + vec2(0.5)
	tl := imageSrc0At(base)
	tr := imageSrc0At(base + vec2(1, 0))
	bl := imageSrc0At(base + vec2(0, 1))
	br := imageSrc0At(base + vec2(1, 1))
	return mix(mix(tl, tr, f.x), mix(bl, br, f.x), f.y)
}
```

Sampling all four taps with `imageSrc0At` gives clamp-to-zero at the image's
edges: within half a pixel of a border, one or two taps fall outside the region
and contribute `vec4(0)`, which shows up as a dark or transparent fringe when
scaling up. For clamp-to-edge or repeat instead, apply the address transform to
each tap, which also makes every tap provably in-region:

```go
func bilinearClampedAt(pos vec2) vec4 {
	p := pos - vec2(0.5)
	f := fract(p)
	base := floor(p) + vec2(0.5)
	tl := imageSrc0UnsafeAt(clampToEdge(base))
	tr := imageSrc0UnsafeAt(clampToEdge(base + vec2(1, 0)))
	bl := imageSrc0UnsafeAt(clampToEdge(base + vec2(0, 1)))
	br := imageSrc0UnsafeAt(clampToEdge(base + vec2(1, 1)))
	return mix(mix(tl, tr, f.x), mix(bl, br, f.x), f.y)
}
```

Substitute `repeat` for `clampToEdge` to get wrapping instead.

Clamping the incoming `pos` once, before the taps are derived, is also a valid
way to get clamp-to-edge — but only against the **texel-centre** range, which
is not what `clampToEdge` computes. That helper clamps to the image region
`[origin, origin+size)`, and a position clamped to `origin` puts the first tap
at `origin-0.5`, outside the image with a real interpolation weight. The range
that works is `[origin+0.5, origin+size-0.5]`:

```go
func clampToTexelCenters(pos vec2) vec2 {
	origin := imageSrc0Origin()
	return clamp(pos, origin+vec2(0.5), origin+imageSrc0Size()-vec2(0.5))
}
```

Interpolation survives in the interior, and at the far edge the second tap of
each pair lands one texel past the last centre with a weight of exactly zero.
Pair input clamping with `imageSrc0At`, whose `vec4(0)` is harmless at weight
zero, rather than `imageSrc0UnsafeAt`, where an undefined value multiplied by
zero need not be zero.

## Pixelation

Constant cell size:

```go
const CellSize = 12.0

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	origin := imageSrc0Origin()
	cell := floor((srcPos-origin)/CellSize)*CellSize + vec2(CellSize/2)
	return imageSrc0At(cell + origin)
}
```

Averaging the whole cell needs a loop, and the loop bound must be constant:

```go
const CellSize = 12.0

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	origin := imageSrc0Origin()
	cell := floor((srcPos-origin)/CellSize) * CellSize
	var acc vec4
	for y := 0.0; y < CellSize; y++ {
		for x := 0.0; x < CellSize; x++ {
			acc += imageSrc0At(cell + vec2(x, y) + origin)
		}
	}
	return acc / (CellSize * CellSize)
}
```

`CellSize` must be integer-valued here, or the loop count and the divisor
disagree.

## A uniform-controlled loop count

Loop to a constant upper bound and `break` on the uniform.

```go
var CellSize float

const MaxCellSize = 32.0

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// Flooring keeps cells on an integer grid: a fractional size puts adjacent
	// cell origins at fractional positions, so neighbouring cells sample
	// overlapping texels under nearest sampling. Clamping keeps the divisor
	// non-zero and the loop bound within the constant's reach.
	size := floor(clamp(CellSize, 1, MaxCellSize))
	origin := imageSrc0Origin()
	cell := floor((srcPos-origin)/size) * size
	var acc vec4
	var n float
	for y := 0.0; y < MaxCellSize; y++ {
		if y >= size {
			break
		}
		for x := 0.0; x < MaxCellSize; x++ {
			if x >= size {
				break
			}
			acc += imageSrc0At(cell + vec2(x, y) + origin)
			n++
		}
	}
	return acc / n
}
```

With `size` quantized, `n` always equals `size*size`; counting the samples is
still worth it because it stays correct if you drop the `floor` to allow a
smooth animated cell size, where the loop runs `ceil(size)` times per axis.
Expect overlapping cells in that case — the grid is no longer aligned to
texels.

## A separable kernel

Offsets in whole pixels need no origin arithmetic, because the origin cancels.

```go
const Radius = 4

func blurH(pos vec2) vec4 {
	var acc vec4
	var weight float
	for i := -Radius; i <= Radius; i++ {
		w := 1.0 - abs(float(i))/(Radius+1)
		acc += imageSrc0At(pos+vec2(float(i), 0)) * w
		weight += w
	}
	return acc / weight
}
```

## Premultiplied alpha

```go
// Fade: scale the whole vector, never the alpha alone. Scaling all four
// components commutes with premultiplication, so this matches what the
// straight-alpha equivalent would produce.
func fade(clr vec4, t float) vec4 {
	return clr * t
}

// Round-trip to straight alpha, needed by any operation that does not commute
// with premultiplication — gamma among them — and by `mix` between samples
// whose alphas differ. Addition needs each input converted as well, but
// converting is not the whole fix: `+` defines no output alpha, so the port
// must also reproduce whatever policy the original used.
func withStraightRGB(clr vec4) vec4 {
	rgb := clr.rgb / max(clr.a, 1e-6)
	rgb = pow(rgb, vec3(1.0/2.2)) // whatever the computation is
	return vec4(rgb*clr.a, clr.a)
}

// A color written as 0..255 channels, premultiplied.
func rgba8(r, g, b, a float) vec4 {
	return vec4(r/255*(a/255), g/255*(a/255), b/255*(a/255), a/255)
}
```

## Branchless selectors

Useful when a branch would diverge widely across a wavefront; for short
branches an ordinary `if` is fine.

```go
func whenGreaterOrEqual(a, b float) float { return step(b, a) }
func whenLess(a, b float) float           { return 1 - step(b, a) }
func whenEqual(a, b float) float          { return 1 - abs(sign(a-b)) }

// Pick a when sel is 0, b when sel is 1.
func pick(a, b vec4, sel float) vec4 {
	return mix(a, b, sel)
}
```

## Debugging

Return a coordinate as a color and look at the result before porting any effect.
A correct destination ramp is black at the destination *image's* top-left, red
across, green down — if the draw covers only part of that image, the ramp spans
the image and the drawn area shows a slice of it.

```go
func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	uv := (dstPos.xy - imageDstOrigin()) / imageDstSize()
	return vec4(uv, 0, 1)
}
```

If the ramp is offset, clipped, or the image only looks right at certain sizes,
an origin is missing somewhere. Substitute `srcPos`/`imageSrc0*` to check the
source side the same way.
