---
name: writing-kage-shaders
description: >
  Use this skill when writing, reviewing, debugging, or porting a Kage shader
  for Ebitengine — anything involving `//kage:unit pixels`, `ebiten.NewShader`,
  `DrawTrianglesShader`, `DrawRectShader`, or a
  `Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4` entry point. Use it
  especially when translating an existing shader from GLSL, Shadertoy, HLSL,
  MSL, Godot, or Unity, where the source assumes 0..1 texture coordinates,
  `gl_FragCoord`, `texture()` sampling, a bottom-left origin, `uniform`
  declarations, a ternary operator, or straight (non-premultiplied) alpha —
  every one of which is wrong in Kage. Covers the atlas-relative coordinate
  model, sampling several source images of different sizes, premultiplied
  alpha, the Go subset Kage actually accepts, and how to compile-check a
  shader without a GPU.
allowed-tools: Read, Edit, Write, Bash
---

# Writing and porting Kage shaders

Kage is Ebitengine's shading language. It looks like Go and is translated at
runtime into GLSL, GLSL ES, HLSL, MSL, or PSSL.

The three things that break ports from every other shading language:

1. **An image is a sub-rectangle of a larger atlas texture.** Coordinates handed
   to and from a Kage shader are absolute positions on that atlas, not positions
   within the image and not `0..1`.
2. **Destination and source images generally live on different atlases**, at
   different origins, so a destination coordinate is not a source coordinate.
3. **Always author in the pixel unit** (`//kage:unit pixels`). The texel unit is
   legacy.

Everything below follows from those.

## Non-negotiables

Check every one of these before declaring a shader done.

- [ ] `//kage:unit pixels` is present, on its own line, exactly once.
- [ ] Every coordinate that gets scaled, divided, rotated, mirrored, tiled, or
      wrapped is made origin-relative first, and the origin is added back after.
- [ ] Every sampling argument is expressed in **source image 0's**
      coordinate space, never image N's.
- [ ] The returned `vec4` is premultiplied alpha (each of `.r/.g/.b` ≤ `.a`).
- [ ] No integer literal division in a float context (`2/3` is `0`).
- [ ] The shader compiles: run the `go test` check in
      [Verifying a shader compiles](#verifying-a-shader-compiles).

## The coordinate model

An `*ebiten.Image` is a region of a bigger internal texture (an atlas). Three
region pairs describe that placement, all in pixels:

| Region | Origin | Size | Whole texture |
|---|---|---|---|
| Destination | `imageDstOrigin()` | `imageDstSize()` | `imageDstTextureSize()` |
| Source N (0..3) | `imageSrcNOrigin()` | `imageSrcNSize()` | `imageSrcNTextureSize()` |

Origin and size are what ordinary pixel-unit sampling uses. The whole-texture
sizes convert between pixels and texels, which is exactly what hand-migrating a
texel-unit calculation needs; `imageSrcNTextureSize` is new in v2.10, and
`imageDstTextureSize`, deprecated in v2.6, is revived there.

These regions describe **whole images**, not the area a draw call covers. The
destination region is the destination image's bounds, so a
`DrawRectShader(w, h, ...)` rectangle placed by a `GeoM` translation, or a
triangle fan covering part of the screen, is a sub-region of it. Consequently
`dstPos.xy - imageDstOrigin()` is image-local, not draw-local, and
`imageDstSize()` is not the draw's extent. When an effect must be positioned or
scaled relative to what is being drawn, drive it from `src0Pos` — which is
interpolated from the vertices and therefore does follow the geometry — or pass
the draw's origin and size as uniforms.

The `Fragment` entry point receives absolute positions on those textures:

```go
func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4
```

- `dstPos.xy` — the fragment's position on the **destination** texture. Only
  `.xy` is meaningful; ignore `.z` and `.w`.
- `src0Pos` — the interpolated position on **source texture 0**. It comes from
  `Vertex.SrcX`/`SrcY`, converted to image 0's texture coordinates. If
  `Images[0]` is nil, `SrcX`/`SrcY` are passed through unconverted.
- `color` — for `DrawTrianglesShader`, the interpolated `Vertex.ColorR/G/B/A`,
  four arbitrary interpolated floats. For `DrawRectShader` it is
  `DrawRectShaderOptions.ColorScale`, which is `vec4(1)` only because a
  zero-value `ColorScale` is (1, 1, 1, 1).
- An optional fourth `custom vec4` parameter receives `Vertex.Custom0..Custom3`
  (`DrawTrianglesShader` only, Ebitengine v2.8+).

Any trailing parameters may be omitted; `func Fragment() vec4` is legal.

### Normalizing to 0..1

Neither `dstPos` nor `src0Pos` is guaranteed to start at zero. Sometimes one
does — an image can sit at its texture's origin, and `DrawTrianglesShader` with
`Images[0] == nil` passes `SrcX`/`SrcY` through unchanged — which is what makes
this the single most common porting bug: code that divides a raw coordinate by
a size looks fine until the image lands elsewhere on an atlas.

```go
// WRONG: dstPos.xy is atlas-absolute, so uv is offset by the atlas origin.
uv := dstPos.xy / imageDstSize()
```

Subtract the origin first, and add it back when converting a normalized value
into a position:

```go
// Destination image, normalized to 0..1.
dstUV := (dstPos.xy - imageDstOrigin()) / imageDstSize()

// Source image 0, normalized to 0..1.
srcUV := (src0Pos - imageSrc0Origin()) / imageSrc0Size()

// Back to a samplable position on source image 0.
pos := srcUV*imageSrc0Size() + imageSrc0Origin()
```

`imageSrc0Size()` is zero when `DrawTrianglesShader` runs with
`Images[0] == nil` — `SrcX`/`SrcY` then reach the shader unconverted and no
source region is set — so normalizing by it divides by zero. Pass the size you
mean as a uniform in that case. `DrawRectShader` is safe without an image: in
the pixel unit it synthesizes source region 0 as the drawn rectangle.

The same subtract-transform-add sandwich applies to *any* non-translation
operation on a coordinate, not just division: `floor(p/cell)*cell` for
pixelation, `mod` for tiling, a rotation matrix, a mirror. Do it in
origin-relative space.

### Sampling

```go
imageSrc0At(pos vec2) vec4                   // returns vec4(0) outside the image region
imageSrc0UnsafeAt(pos vec2) vec4             // faster; undefined outside the image region
imageSrcNAtFromSrc0Pos(pos vec2) vec4        // N >= 1; returns vec4(0) outside the image region
imageSrcNUnsafeAtFromSrc0Pos(pos vec2) vec4  // N >= 1; faster; undefined outside the image region
```

Sampling is always nearest-neighbour. There is no linear filter, no mipmap, no
wrap mode inside a shader; `ebiten.Filter` and `ebiten.Address` apply to
`DrawImage`/`DrawTriangles`, not to a custom shader. Implement clamping,
repeating, and interpolation yourself — see
[reference/recipes.md](reference/recipes.md).

The `UnsafeAt` forms omit that region bounds check, so their result outside the
source region is undefined: depending on the position and the backend they may
return atlas padding, a neighbouring image's pixels, zero, or something else
entirely. Use them only where the position is provably inside the region — after
a clamp or a wrap, for instance.

## Sampling more than one source image

For N ≥ 1 the functions are `imageSrcNAtFromSrc0Pos` and
`imageSrcNUnsafeAtFromSrc0Pos`, and their names state the rule: they **take a
position in source image 0's coordinate space**. The implementation rebases the
position: it fetches `pos - imageSrc0Origin() + imageSrcNOrigin()`, and
`imageSrcNAtFromSrc0Pos` returns `vec4(0)` outside
`[imageSrc0Origin(), imageSrc0Origin() + imageSrcNSize())`. (`imageSrcNAt` and
`imageSrcNUnsafeAt` are the same functions under their old names, deprecated as
of v2.10.)

So all four slots share one coordinate space, and the normal thing to write is
`src0Pos` for every one of them. Pixel (x, y) of image 0 lines up with pixel
(x, y) of image N, whatever atlases they landed on and whatever their sizes:

```go
return imageSrc0At(src0Pos) * imageSrc1AtFromSrc0Pos(src0Pos).a
```

Differing sizes need no correction. Where image N is smaller than image 0,
`imageSrcNAtFromSrc0Pos` just returns `vec4(0)`.

Arithmetic is needed only when you deliberately want a *different* mapping, such
as stretching a smaller mask across the whole of image 0. That is a scaling
choice, not a size fix, and the origin you add back is still image 0's:

```go
uv := (src0Pos - imageSrc0Origin()) / imageSrc0Size()
src0PosForStretchedSrc1 := uv*imageSrc1Size() + imageSrc0Origin() // note: Src0Origin, not Src1Origin
return imageSrc1AtFromSrc0Pos(src0PosForStretchedSrc1)
```

Adding `imageSrc1Origin()` there is the classic bug: the origin is applied
internally, so adding it again double-counts the atlas offset and samples an
unrelated part of the atlas. `imageSrcNOrigin()` for N ≥ 1 is almost never what
you want in a sampling expression; `imageSrc0Origin()` is the origin that
matters.

## Always use the pixel unit

Put this on its own line, conventionally just above `package main`:

```go
//kage:unit pixels
```

Omitting it selects the legacy texel unit, in which `src0Pos` and every
origin/size value become texels of the **atlas** texture (0..1 across the whole
atlas, not across your image), and `DrawTrianglesShader`/`DrawRectShader` panic
when the source images differ in size. A file may contain at most one
`//kage:unit` directive.

The texel-unit helpers `imageSrcRegionOnTexture()`, `imageDstRegionOnTexture()`,
and the index-less `imageSrcTextureSize()` are deprecated as of v2.6. Do not
introduce them, and replace them when porting old Kage code:

| Deprecated | Replacement |
|---|---|
| `imageSrcRegionOnTexture()` | `imageSrc0Origin()`, `imageSrc0Size()` |
| `imageDstRegionOnTexture()` | `imageDstOrigin()`, `imageDstSize()` |
| `imageSrcTextureSize()` | `imageSrc0TextureSize()` (v2.10+) |

## Premultiplied alpha

Ebitengine works in premultiplied alpha everywhere, shaders included.

- Sampling returns premultiplied colors.
- The `vec4` you return must be premultiplied: `rgb ≤ a` component-wise.
  Nothing clamps it for you; an invalid color renders as a too-bright,
  washed-out blend.
- To fade, scale the whole vector: `return clr * 0.5`, never `clr.a *= 0.5`.
- To tint, multiply: `return clr * vec4(1, 0.5, 0.5, 1)`.
- A shader ported from a straight-alpha source needs its final color
  premultiplied: `return vec4(rgb*a, a)`. Fixing only the output is not enough
  if the original also computed on straight-alpha samples, since sampling
  always hands you premultiplied ones. Whether the original's were straight
  depends on its texture data and its own conventions, not on the language it
  was written in.
- An RGB operation may stay in premultiplied space when it commutes with
  premultiplication: when `F(a*rgb) == a*F(rgb)` across the alpha range in
  play. Scaling the whole `vec4` (a fade) qualifies — it leaves the straight
  RGB untouched and scales only alpha — as does a constant per-channel tint
  with an alpha factor of 1.
- `mix` does not, despite looking linear. Over premultiplied values it
  interpolates in premultiplied space, which is not the straight-RGB result the
  original computed — unless the two alphas are equal, in which case the two
  agree. It is not source-over compositing either: `mix` is interpolation, not
  a blend operator.
- `+` has no equal-alpha exception, because addition carries no output-alpha
  policy of its own. Summing two premultiplied `vec4` values also sums their
  alphas, and the straight RGB that comes back out is scaled by that larger
  alpha: two colors at `a = 0.5` add to `a = 1` with straight RGB at half the
  straight-space sum. Whether the original preserved, maximized, clamped, or
  separately computed its output alpha has to be read off the original and
  reproduced deliberately.
- Nonlinearity is not itself the criterion. Common failures are gamma
  correction, contrast curves with a fixed pivot, most `pow` exponents, and
  thresholds against a fixed nonzero value. A positively homogeneous
  operation — a per-channel `min`/`max` against zero, say — commutes even
  though it is not linear.
- HSV and HSL must be judged case by case against `F(a*rgb) == a*F(rgb)` rather
  than assumed to fail, and the two do not behave alike. A conversion followed
  by its own inverse is the identity in both, so it commutes. Reshaping value
  or lightness generally does not.
- In HSV, saturation is `(max-min)/max` and hue is a ratio, so both are
  invariant under positive scaling while value alone carries the scale. An
  operation touching only hue and saturation commutes, as long as the concrete
  implementation introduces no offset and no fixed-value step.
- In HSL, saturation is `(max-min)/(max+min)` below `L = 0.5` and
  `(max-min)/(2-max-min)` above it. The first form is scale-invariant; the fixed
  white point makes the second one scale-dependent, and scaling can carry a
  color across the boundary between them. Multiplying saturation by a constant
  does commute, since that white point cancels between the forward and inverse
  formulas — but only while the whole operation stays homogeneous, with no
  offset, no fixed assignment, and no clamp. The usual `min(s*k, 1)` is enough
  to break it, whenever the straight-space saturation reaches the limit and the
  one derived from premultiplied RGB does not. A saturation curve or a fixed
  saturation does not commute either. Evaluate each HSL operation on its own,
  and unpremultiply when you have not.
- To convert, unpremultiply, compute, premultiply again, guarding the divide:
  `rgb := clr.rgb / max(clr.a, 1e-6)`. Convert whenever you have not
  established equivalence.

Worked through, for a straight `(1, 0, 0)` at `a = 1` and a straight
`(0, 0, 1)` at `a = 0.25`. Note that `(0, 0, 1, 0.25)` is not a legal
premultiplied color at all, since blue exceeds alpha:

| | first | second | `mix(…, 0.5)` |
|---|---|---|---|
| straight | `(1,0,0) a=1` | `(0,0,1) a=0.25` | `(0.5,0,0.5) a=0.625` |
| premultiplied | `(1,0,0,1)` | `(0,0,0.25,0.25)` | `(0.5,0,0.125,0.625)` |

The premultiplied result unpremultiplies to straight `(0.8, 0, 0.2)`; the
straight result premultiplies to `(0.3125, 0, 0.3125, 0.625)`. The two disagree
precisely because the alphas differ.

## What Kage is not

Kage is a small subset of Go's syntax, not Go and not GLSL. Verified against
this repository:

**Available:** `bool`, `int`, `float`, `vec2`/`vec3`/`vec4`,
`ivec2`/`ivec3`/`ivec4`, `mat2`/`mat3`/`mat4` (column-major), fixed-size arrays;
swizzling (`.xyzw`, `.rgba`, `.stpq`, not mixed) and `[i]` indexing; top-level
helper functions with multiple return values; `const`; `if`/`else`;
`break`/`continue`; the full builtin math library; `discard()`;
`frontfacing()` (v2.9+).

**Not available:** `struct`, `switch`, `goto`, `defer`, `go`, `init`, `import`,
methods, nested functions and closures, strings, slices, maps, pointers.
Recursion passes Kage's own compiler but the target shading languages reject it,
so it fails at draw time — do not use it.

**Rules that bite:**

- **Loops are constant-bounded.** The only accepted forms are
  `for i := <const>; i <op> <const>; i <op>= <const> { }` (ops `< <= > >= == !=`
  and `+= -= ++ --`) and, as of v2.10, `for i := range <constant int>` and
  `for i, v := range <array>`. There is no `for cond { }` and no
  uniform-controlled bound. To vary the count at runtime, loop to a constant
  upper bound and `break` on a uniform.
- **Globals must be exported, and exported globals are uniforms.** `var Time
  float` declares a uniform; `var t float` is a compile error. Uniforms cannot
  be assigned in the shader and cannot have initial values. `const` globals may
  be any case.
- **No implicit int↔float conversion**, exactly as in Go. `var f float = i`
  fails; write `float(i)`. `%` is not defined on floats; use `mod(x, y)`.
- **`2/3` is `0`.** Untyped constant division follows Go's rules, so
  `color.r >= 2/3` silently compares against zero. Write `2.0/3.0`.
- **`float` has no guaranteed precision.** It is the target language's
  `float` at its highest available precision, which on GLES is whatever the
  driver gives for `highp`. Do not rely on a specific mantissa width, and do
  not port a shader that packs data into float bits.
- **`len` takes an array only**, unlike the Go-ish `len(vec4)` some third-party
  docs suggest.
- **There are exactly 4 source images**, `imageSrc0*` through `imageSrc3*`.
- **There is no vertex entry point.** Kage compiles a fragment shader; vertex
  transformation is Ebitengine's. A function you name `Vertex` is just an
  ordinary helper and is never called.

## Migrating an existing shader

1. **Read the original for its coordinate assumptions first.** Note whether it
   uses `0..1` UVs, pixel coordinates, a bottom-left origin (Shadertoy, OpenGL)
   or top-left (Ebitengine, Direct3D), and whether it wants a resolution or
   aspect-ratio uniform.
2. **Translate the coordinate preamble, not just the body.** Replace the
   original's UV derivation with the origin-relative form above.
   `gl_FragCoord`/`fragCoord` becomes `dstPos.xy - imageDstOrigin()` and
   `iResolution` becomes `imageDstSize()` *when the draw covers the whole
   destination image*; otherwise work from `src0Pos`/`imageSrc0Size()`, or pass
   the draw's origin and size as uniforms. A bottom-left origin needs
   `uv.y = 1 - uv.y`.
3. **Translate the body** using the table in
   [reference/porting-to-kage.md](reference/porting-to-kage.md).
4. **Rewrite unbounded loops** to a constant bound with a `break`.
5. **Fix the output** to premultiplied alpha.
6. **Move `#define`s to `const`, `uniform`s to exported globals.**
7. **Compile-check**, then render and compare against the original.

Port the coordinate handling first and verify it in isolation (return
`vec4(dstUV, 0, 1)` and confirm a clean red/green ramp across the image) before
porting the effect. Almost every "the effect is offset / tiled wrong / only
works when the image happens to be the whole atlas" bug is a missing origin.

## Verifying a shader compiles

`ebiten.NewShader` compiles Kage with no GPU, window, or display, so a plain
`go test` is the fastest check:

```go
package shaders_test

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestShaderCompiles(t *testing.T) {
	src, err := os.ReadFile("effect.kage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ebiten.NewShader(src); err != nil {
		t.Fatal(err)
	}
}
```

Errors are reported as `line:column: message` against the source.

Two limits to keep in mind:

- This validates Kage only. Translation to GLSL/HLSL/MSL happens in the
  graphics driver on first use, so backend-level rejections (recursion,
  driver-specific limits) surface at draw time, not here.
- It cannot catch a wrong result. For that, render with
  [skills/run-ebitengine-app-headless](../run-ebitengine-app-headless/SKILL.md)
  and compare pixels.

A real `.kage` file needs no build constraint; the Go toolchain ignores the
extension. `//go:build ignore` above the `//kage:unit` directive is only for
Kage source kept with a `.go` extension so that Go tooling skips it, which is
what `examples/shader/*.go` in this repository does.

## Reference

- [reference/porting-to-kage.md](reference/porting-to-kage.md) — construct-by-construct
  translation table for GLSL, Shadertoy, HLSL, and Godot, plus the CPU-side
  invocation these ports need.
- [reference/recipes.md](reference/recipes.md) — copy-paste Kage snippets:
  clamp/repeat addressing, bilinear sampling, normalization helpers,
  cross-image remapping, pixelation, blur, branchless selectors.
- <https://ebitengine.org/en/documents/shader.html> — the official language
  reference, including the complete builtin function list.
- <https://github.com/tinne26/kage-desk/tree/main/docs> — tutorials and further
  pitfalls.
