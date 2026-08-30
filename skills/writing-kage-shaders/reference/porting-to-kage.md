# GLSL / Shadertoy / HLSL → Kage

Companion to [../SKILL.md](../SKILL.md). Read the coordinate model there first;
this file is the mechanical translation layer.

## Program structure

| Source | Kage |
|---|---|
| `#version 330` etc. | nothing; drop it |
| a vertex shader | nothing; Ebitengine owns vertex transformation |
| `uniform float iTime;` | `var Time float` (exported global = uniform) |
| `#define PI 3.14159` | `const PI = 3.14159` |
| `varying` / `in` / `out` | `Fragment` parameters and the return value |
| `void mainImage(out vec4 o, in vec2 c)` | `func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4` |
| `gl_FragColor = c;` / `fragColor = c;` | `return c` |
| `struct Ray { ... }` | not supported; pass components separately or use an array |
| `#ifdef` / `#include` | not supported; no preprocessor at all |

## Coordinates

| Source | Kage |
|---|---|
| `gl_FragCoord.xy` (window pixels) | `dstPos.xy - imageDstOrigin()` † |
| `iResolution.xy` | `imageDstSize()` † |
| `fragCoord/iResolution.xy` (0..1 uv) | `(dstPos.xy - imageDstOrigin()) / imageDstSize()` † |
| `vTexCoord` / `v_texCoord` (0..1 uv) | `(src0Pos - imageSrc0Origin()) / imageSrc0Size()` |
| `SCREEN_UV`, `UV` (Godot) | as above, destination and source respectively |
| bottom-left origin | Ebitengine's is top-left: `uv.y = 1 - uv.y` |
| `gl_FrontFacing` | `frontfacing()` (v2.9+) |
| `discard;` | `discard()` |

† These hold when the draw covers the whole destination image. The destination
region is the destination *image*, not the rectangle or triangles being drawn,
so under a `GeoM` translation or a partial-coverage draw they give image-local
coordinates and the image's size, not the draw's. For a partial draw, derive the
coordinates from `src0Pos`, which is interpolated from the vertices, or pass the
draw's origin and size as uniforms.

Aspect correction that reads `uv = (fragCoord - 0.5*iResolution.xy)/iResolution.y`
translates directly once `fragCoord` and `iResolution` are replaced, subject to
the same caveat: both sides are then in image-local pixels.

## Sampling

| Source | Kage |
|---|---|
| `texture(iChannel0, uv)` / `texture2D(tex, uv)` | `imageSrc0At(uv*imageSrc0Size() + imageSrc0Origin())` |
| `texture(tex, vTexCoord)` where uv is the fragment's own uv | `imageSrc0At(src0Pos)` |
| a second sampler at the same uv | `imageSrc1AtFromSrc0Pos(src0Pos)` ‡ — any size; every slot shares image 0's space |
| a second sampler stretched over image 0 | `imageSrc1AtFromSrc0Pos(uv*imageSrc1Size() + imageSrc0Origin())` ‡ |
| `textureSize(tex, 0)` | `imageSrc0Size()` (the image; `imageSrc0TextureSize()` is the atlas) |
| `texelFetch(tex, ivec2(x, y), 0)` | `imageSrc0At(vec2(x, y) + imageSrc0Origin())` |
| `textureLod`, mipmaps, `textureGrad` | not available |
| `GL_CLAMP_TO_EDGE` / `GL_REPEAT` sampler state | implement in-shader; see [recipes.md](recipes.md) |
| linear filtering | implement in-shader; sampling is always nearest |
| out-of-range sampling | the `At` forms give `vec4(0)`; the `UnsafeAt` forms are undefined |

‡ `imageSrcNAtFromSrc0Pos` and `imageSrcNUnsafeAtFromSrc0Pos` (N ≥ 1) are new in
v2.10. Before that, use `imageSrcNAt` and `imageSrcNUnsafeAt`: the same functions
under their old names, still working but deprecated.

Offsetting a sample by whole pixels needs no origin arithmetic, because the
origin cancels: `imageSrc0At(src0Pos + vec2(1, 0))` is the neighbouring texel.
Only *scaling* a coordinate needs the subtract-transform-add sandwich.

## Builtin functions

Same name, same behaviour: `sin cos tan asin acos atan pow exp log exp2 log2
sqrt inversesqrt abs sign floor ceil fract mod min max clamp mix step smoothstep
length distance dot cross normalize faceforward reflect refract transpose fwidth`.

| Source | Kage |
|---|---|
| `atan(y, x)` (two-argument) | `atan2(y, x)` — same order |
| `dFdx` / `dFdy` | `dfdx` / `dfdy` |
| `radians(d)` / `degrees(r)` | not available; `d*3.14159265/180`, `r*180/3.14159265` |
| `trunc`, `round`, `roundEven`, `modf` | not available; compose from `floor`/`ceil`/`sign`/`fract` |
| `sinh`, `cosh`, `tanh` | not available; write them out |
| `inverse`, `determinant`, `outerProduct`, `matrixCompMult` | not available |
| `lessThan`, `greaterThan`, `all`, `any`, `not` | not available; compare components or use `step` |
| `isnan`, `isinf` | not available |
| `floatBitsToInt`, `packHalf2x16`, bitfield ops | not available |
| `mix(a, b, bvec)` | not available; `mix` takes a float/vector `t` only |
| `x % y` on floats | `mod(x, y)` (`%` is integer-only in Kage) |
| `cond ? a : b` | `if`/`else`, or `mix(b, a, step(...))` |
| `float x = intExpr;` | `x := float(intExpr)` — no implicit conversion |

`length`, `dot`, `normalize` and friends work on `vec2`/`vec3`/`vec4` as in
GLSL. `cross` is `vec3`-only.

## Control flow

| Source | Kage |
|---|---|
| `for (int i = 0; i < N; i++)` with constant `N` | `for i := 0; i < N; i++ { }` |
| `for (...)` with a uniform bound | constant upper bound plus `if i >= Bound { break }` |
| `while (cond)` / `do { } while` | not available; rewrite as a bounded `for` with `break` |
| `switch` | `if`/`else if` chain |
| recursion | not available in the target languages; flatten it |

The classic Kage loop must be literally
`for <var> := <const>; <var> <op> <const>; <var> <op>= <const> { }`. The bounds
and the step are constants, not uniforms and not computed values.

## Types

| Source | Kage |
|---|---|
| `float`, `int`, `bool` | same; `float`'s precision is not guaranteed |
| `vec2/3/4`, `ivec2/3/4`, `mat2/3/4` | same; matrices are column-major |
| `bvec2/3/4`, `uint`, `uvec*`, `double` | not available |
| `sampler2D` as a variable or parameter | not available; images are fixed slots 0..3 |
| `vec4 a[4];` | `var a [4]vec4` |
| arrays of samplers, dynamic sampler indexing | not available |

Swizzles work the same (`.xyzw`, `.rgba`, `.stpq`), and groups may not be mixed.

## Output color

Ebitengine expects **premultiplied alpha**. A shader written for straight alpha
ends with something like `fragColor = vec4(rgb, a);` — that becomes:

```go
return vec4(rgb*a, a)
```

Anything that returns `.rgb` unscaled while varying `.a` is a straight-alpha
shader and will look wrong when composited.

The inputs need the same attention. Sampling always returns premultiplied
samples. GLSL's `texture()` carries no alpha convention — it returns the stored
texel — so what matters is what the original's texture data and calculations
assumed. If they assumed straight alpha, convert before any straight-RGB
calculation: `rgb := clr.rgb / max(clr.a, 1e-6)`, then premultiply the result
on the way out.

Only operations that commute with premultiplication can skip the conversion —
those where `F(a*rgb) == a*F(rgb)` over the alpha range in play. Scaling the
whole `vec4` qualifies, as does a constant per-channel tint with an alpha factor
of 1. `mix` does not: over premultiplied colors it interpolates in
premultiplied space, which differs from the original's straight-RGB result
whenever the alphas differ (and agrees when they are equal). `+` has no such
exception — summing premultiplied values sums the alphas too, so it must be
reproduced together with whatever output-alpha policy the original used.
Nonlinearity is not itself the criterion, but common failures include gamma,
contrast curves with a fixed pivot, most `pow` exponents, and thresholds
against a fixed nonzero value. HSV and HSL must be judged individually, and do
not behave alike. In HSV, hue and saturation are invariant under positive
scaling and value alone carries the scale, so an operation on hue and saturation
commutes provided it introduces no offset and no fixed-value step. In HSL,
saturation is `(max-min)/(max+min)` below `L = 0.5` and `(max-min)/(2-max-min)`
above it, and the fixed white point in that second form makes it scale-dependent.
Multiplying saturation by a constant commutes only while the operation stays
homogeneous — no offset, no fixed assignment, no clamp, so the usual
`min(s*k, 1)` is enough to break it — and a saturation curve or a fixed
saturation does not commute at all. Reshaping value or lightness generally does
not either. Convert unless you have established equivalence.

## Shadertoy specifics

| Shadertoy | Kage |
|---|---|
| `iResolution` | `imageDstSize()` (`vec2`; there is no `.z`), if the draw covers the whole image |
| `iTime`, `iTimeDelta`, `iFrame` | your own uniforms, updated per tick |
| `iMouse` | your own `vec2`/`vec4` uniform from `ebiten.CursorPosition` |
| `iChannel0..3` | source images 0..3 |
| `iChannelResolution[N].xy` | `imageSrcNSize()` |
| `iDate`, `iSampleRate`, `iChannelTime` | your own uniforms |
| bottom-left origin | flip: `uv.y = 1 - uv.y` |
| `mainImage(out fragColor, in fragCoord)` | `Fragment` returning the color |
| cubemaps, 3D textures, buffers, sound shaders | not supported |

Shadertoy shaders are opaque by convention (`fragColor.a` is often left at 1),
so a direct port usually needs no alpha work — but if the effect fades, apply
the premultiplication.

## HLSL / Unity specifics

| HLSL | Kage |
|---|---|
| `float2/3/4`, `int2/3/4` | `vec2/3/4`, `ivec2/3/4` |
| `float4x4` | `mat4`; for the uniform layout see **The CPU side** below |
| `tex2D(s, uv)` / `t.Sample(s, uv)` | `imageSrc0At(...)` |
| `lerp`, `frac`, `saturate`, `atan2`, `ddx`, `ddy` | `mix`, `fract`, `clamp(x, 0, 1)`, `atan2`, `dfdx`, `dfdy` |
| `mul(M, v)` | `M * v` |
| `mul(v, M)` | `v * M` |
| `SV_Position` | `dstPos` |
| `cbuffer` fields | individual exported globals |
| `[unroll]`, `[loop]`, semantics, register bindings | drop them |

Do not transpose a matrix just because the source is HLSL. HLSL storage may be
row-major or column-major depending on declarations and compiler flags; what
matters is the order of the flat array your CPU code already builds. Reorder it
only if it is row-major. Multiplication is a separate question from storage,
and the `mul` rows above cover it.

## The CPU side

Uniform values are set through the draw options. Values must be a
numeric/boolean type or a slice/array of one, flattened linearly: a `vec4` is 4
floats, a `mat4` is 16, a `[4]vec4` is 16. A uniform not present in the map is
treated as zero. A wrong length or type panics.

Matrices are flattened **column-major** — one column after another. A `mat2`
whose logical rows are `[[a, b], [c, d]]` is supplied as
`[]float32{a, c, b, d}`; `mat3` and `mat4` follow the same column-by-column
rule. Use this layout whatever the active graphics backend is: Ebitengine
performs the DirectX conversion internally, so never transpose based on which
backend the game happens to be running on.

### `DrawRectShader`

Simplest, and enough for a full-screen or full-image effect. Every non-nil
source image must be exactly `width`×`height`, and `src0Pos` runs across that
rectangle — including when `Images[0]` is nil, since the pixel unit synthesizes
source region 0 from the rectangle.

```go
op := &ebiten.DrawRectShaderOptions{}
op.Images[0] = src
op.Uniforms = map[string]any{
	"Time": float32(t),
}
dst.DrawRectShader(src.Bounds().Dx(), src.Bounds().Dy(), shader, op)
```

### `DrawTrianglesShader`

Needed whenever source images differ in size from each other or from the
destination area, or the geometry is not an axis-aligned rectangle. Set both
`DstX`/`DstY` and `SrcX`/`SrcY`; `SrcX`/`SrcY` are in source image 0's own
bounds, so a sub-image does not start at `(0, 0)`.

```go
db := dst.Bounds()
sb := src.Bounds()
vs := []ebiten.Vertex{
	{DstX: float32(db.Min.X), DstY: float32(db.Min.Y), SrcX: float32(sb.Min.X), SrcY: float32(sb.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	{DstX: float32(db.Max.X), DstY: float32(db.Min.Y), SrcX: float32(sb.Max.X), SrcY: float32(sb.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	{DstX: float32(db.Min.X), DstY: float32(db.Max.Y), SrcX: float32(sb.Min.X), SrcY: float32(sb.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	{DstX: float32(db.Max.X), DstY: float32(db.Max.Y), SrcX: float32(sb.Max.X), SrcY: float32(sb.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
}
is := []uint16{0, 1, 2, 1, 2, 3}

op := &ebiten.DrawTrianglesShaderOptions{}
op.Images[0] = src
op.Images[1] = mask
op.Uniforms = map[string]any{"Time": float32(t)}
dst.DrawTrianglesShader(vs, is, shader, op)
```

`ColorR/G/B/A` reach the shader as the `color vec4` parameter, interpolated and
uninterpreted — with `DrawTrianglesShader` they are four arbitrary floats, not a
color multiplier. With `DrawRectShader` that parameter carries
`DrawRectShaderOptions.ColorScale`, which is `vec4(1)` only when left at its
zero value. `Custom0..Custom3` reach an optional fourth `custom vec4`
parameter (v2.8+).

Compile the shader once and reuse it; `ebiten.NewShader` is not something to
call per frame.
