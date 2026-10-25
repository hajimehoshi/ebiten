# OpenType SVG

Internal glyph interpretation and GPU rendering for
[#3796](https://github.com/hajimehoshi/ebiten/issues/3796). Document parsing,
glyph selection, retained geometry, viewport mapping, styles and paint
interpretation remain CPU-only. `PrepareRender` produces a bounded render plan;
`svgrender.Renderer` uses vector paths and `vector.ImagePaint` for gradient paints. Normal `text/v2` drawing and glyph-image access use this renderer. Successful
parsing alone does not establish rendering support.

[Acceptance fixtures](testdata/README.md) describe the real-font records,
focused cases, licenses and intended rendering results. Existing
[oksvg tests](../oksvg/oksvg_test.go) and
[glyph-document tests](../../gotextsvgdoc_test.go) are compatibility inputs,
not specifications for new behavior.

## Supported behavior

The target is the required subset of [OpenType SVG][opentype] and SVG 1.1.

| Area | Implemented |
| --- | --- |
| Documents | UTF-8 XML, glyph IDs, shared definitions, local references, bounded use traversal |
| Geometry | Paths, basic shapes, transforms, lengths and viewport mapping |
| Styles | Presentation attributes, inline styles, the CSS subset below, cascade and inheritance |
| Paint | Named/numeric colors, currentColor, CPAL variables, local URL paints and fallbacks |
| Gradients | Linear/radial definitions, href inheritance, stops, units, transforms, spread and focal parameters |
| Rendering | Painter order, transformed fills/strokes, dashes, gradient sampling and nested clips |
| Composition | Paint opacity, display/visibility and isolated element/group opacity |
| Images | Embedded PNG/JPEG data URIs, viewport mapping, transforms and clips |

The root viewport preserves font coordinates without introducing an implicit
clip. Style and gradient lengths enforce the OpenType restriction on em/ex.

Selectors use source ancestry, while inherited values come from the explicit
instance parent. This follows [SVG 1.1 use semantics][svg-use] for glyph
selection. Stylesheets outside the selected glyph still apply. Missing glyph
IDs return `errGlyphNotFound`; whole-document fallback is permitted only when
no glyph ID exists anywhere in the document. Duplicate IDs select the first
retained element in document order.

### CSS subset and policy

[svgcss](svgcss) parses syntax only. Document matching, cascade, inheritance,
SVG property interpretation and palette resolution remain in this package.

Supported selectors are ASCII type names, universal selectors, IDs, classes,
compounds, descendant/child combinators and selector lists. CSS property and
function names are case insensitive; IDs and classes are case sensitive.
Author importance, specificity and declaration order determine the cascade.
Class-token order and repeated class names do not change the result.

Comments between tokens and quoted local URLs are accepted. Escaped/non-ASCII
identifiers, comments splitting tokens, namespace/attribute/pseudo/sibling
selectors, at-rules and custom-property definitions are unsupported. Style
media may be empty, all or screen; type may be empty or text/css with MIME
parameters. Other media/type values are unsupported.

Unknown properties are ignored. Invalid declarations are discarded before the
cascade; broken stylesheet delimiters, quotes or comments return `errStyle`.
Known deferred effects report `errUnsupported` only when effective. Overridden
or unmatched declarations and display:none instances do not reject artwork.
The property table and validators in [css.go](css.go) and [style.go](style.go)
define the exact property vocabulary, initial values and applicability rules.

Implemented properties cover fill/stroke paints, fill/clip rules, stroke width,
caps, joins, miter limit and dashes, color, paint/stop/compositing opacity,
stop color, clip-path references, display, visibility and color interpolation.
Explicit inherit, initial and unset are supported. CSS transforms and geometry
properties are deferred, including initial values that override XML geometry.
Their ordinary XML attributes remain inputs to the geometry parser.

Filters, patterns, effective mask images, markers and other deferred rendering
effects are reported rather than silently omitted. Inert mask longhands are
ignored. External paint URLs are unsupported and are never fetched. Restricted
OpenType element subtrees and animation subtrees are ignored by the document
parser. Static rendering never runs animations. Unknown rendered
elements report `ErrUnsupported`. Nested SVG viewports,
`symbol` instances and pathLength-calibrated dashes are currently unsupported.

### Compatibility allowances

Colors support SVG named colors, hexadecimal RGB, numerical/percentage RGB,
currentColor, none, local URL fallbacks and CPAL var(--colorN, fallback).
Variable fallbacks are evaluated only when needed; invalid results fall back
to inherited/initial values at computed-value time. Palette alpha is applied
once without altering the opacity inherited by children.

HSL and mixed numeric/percentage RGB preserve oksvg compatibility. RGBA, HSLA,
transparent and percentage opacity are additional allowances. Other allowances
include empty SVG namespaces, unprefixed href, adjacent transform functions,
repeated transform separators, and a zero gradient focal radius. These do not
imply general CSS, SVG 2 or browser rendering support.

## Rendering

`PrepareRender(source, RenderOptions)` is CPU-only. Supply the glyph ID, UPEM,
output dimensions, font-to-output transform, foreground/palette and sample
density (1, 2, 4 or 8 per axis). Foreground is straight `color.NRGBA`; use opaque
black for the ordinary default. A zero transform means identity. Treat the
returned plan as immutable.

Call `svgrender.Renderer.Draw` and `Deallocate` within Ebitengine's graphics
lifecycle. Reuse a renderer for successive glyphs. Gradient paints are rasterized
on the chosen grid and applied through `vector.ImagePaint`. Normal text drawing
sets `Samples: 1, AntiAlias: true`: vector antialiases each path and its clip at
the glyph's native resolution. The completed surface is copied directly into
the cached glyph image, with no supersampling or downsampling pass. The internal
point-sample mode (`AntiAlias: false`) remains available for reference tests;
its samples are averaged only after composition.

Empty gradients disable paint; single-stop and degenerate linear/radial
gradients use the final stop. Singular gradient transforms use a supplied URL
fallback or return `ErrGeometry`; excessive inverse scales return `ErrLimit`.
At a radial focus on the circumference, rays without a forward circle
intersection take the final stop, including with repeat/reflect. Singular
geometry transforms draw nothing. Nonfinite values and excessive coordinates
fail preparation.

Clip children form a union; own and ancestor clips intersect that union.
Descendants of hidden elements may set `visibility:visible`. display:none
subtrees are pruned while shared definitions remain available. Display does not
apply to a clipPath itself, but does apply to its children. Shape elements have
no rendered children. Clip content is limited to shapes and uses that directly
reference a shape; groups and indirect use chains are ignored. Malformed
geometry that must be interpreted rejects the whole plan, including any valid
prefix. Errors never return partial successful artwork;
use `errors.Is` with the exported error identities to classify failures.

Zero opacity produces no draws and skips painting effects. Such a subtree is
otherwise skipped unless an active ancestor needs its geometry for an
objectBoundingBox clip. In that case its geometry, transforms, display values
and use references contribute to the bounds; errors needed to resolve those
bounds propagate. Filters, masks and other painting effects remain ignored.
Opacity is ignored in clip content.
Opacity on an element or group applies once to its completed paints and children.
Each translucent element owns an isolated transparent surface at its parent's
resolution. Inherited fill/stroke opacity still applies independently to each
paint. Group boundaries flush pending vector work before composition.

With native vector AA, a path and its clip share coverage, but separate draws
are composited after their coverage has been resolved. Consequently, coincident
opaque overpainting can leave a fringe of the underlying paint, and adjoining
fills can leave a partial-coverage seam. Isolated opacity groups do not remove
these edge artifacts. This is an explicit limitation of the native-AA baseline,
not a claim of sample-preserving SVG composition. The reference point-sample
mode retains coverage through composition and continues to test those stronger
invariants. Further composition work must preserve the native-size policy.

Images accept only embedded `data:image/png` and `data:image/jpeg` URIs,
with base64 or percent-encoded bytes. External URLs and local files are never
loaded. MIME types must match the decoded format. Malformed image data returns
`ErrImage`, unsupported formats/locations return `ErrUnsupported`, and exhausted
budgets return `ErrLimit`. Zero-sized, hidden and zero-opacity images do not
decode; their viewport geometry contributes to object bounds when needed.
Image viewports support x/y/width/height lengths, preserveAspectRatio alignment,
meet/slice/none, transforms and clipping. Fill and stroke do not paint images.

Image texels are sampled with nearest-neighbor filtering. Native rendering
uses vector AA for viewport/clip edges; point-sample reference rendering uses
its final full-sample average. This preserves hard texel boundaries and premultiplied alpha; it is not a smooth
reconstruction filter or a production minification policy. Decoding and conversion
to premultiplied RGBA are CPU-only. Identical URI strings share decoded assets
within a plan; each asset is uploaded once per Draw and released when Draw ends.
The text integration caches final glyph images; embedded-image GPU uploads
are not retained across plans.

Full-font and platform acceptance, quality/performance evaluation, and removal
of the unused oksvg implementation and its dependencies remain separate work.

## Limits

Inputs are bounded: decoded documents allow 32 MiB, CSS text 1 MiB, and
individual SVG property/attribute values 16 KiB. The CSS parser independently
limits value text to 16 KiB; ParseValue excludes priority and surrounding
whitespace from that limit. Raw declaration input remains bounded by 1 MiB.

Depth, element, rule, selector, stop and dash limits are declared beside their
parsers in [document.go](document.go), [number.go](number.go),
[css.go](css.go) and [svgcss/syntax.go](svgcss/syntax.go). Resolver work,
instances and cached values have independent budgets; expensive input can hit
`ErrLimit` before another ceiling is reached. These are record/work bounds,
not allocator byte guarantees.

Render preparation bounds geometry, instances, gradient stops, stroke expansion,
coordinates and sample surfaces. Isolation allows at most 16 simultaneous group
levels and 16,777,216 group sample pixels (64 MiB of requested RGBA regions).
The renderer reuses one full-size surface per depth, releases excess surfaces on
the next nonempty Draw, and releases all resources in Deallocate. Full-size
surfaces conservatively preserve one origin and grid; small groups do not get
tightly cropped allocations. Group clears and composites each charge their full
area against the existing 268,435,456-sample cumulative work budget. Root and
gradient scratch surfaces are separately bounded by the output sample limits.

Embedded images allow at most 256 distinct URI assets, 4 MiB per encoded URI,
16 MiB of aggregate URI bytes, 4096 pixels per axis, 4,194,304 pixels per image,
and 8,388,608 aggregate decoded pixels (32 MiB of retained RGBA). These limits
also bound uploads per Draw. DecodeConfig checks dimensions and pixel budgets
before Decode. Decoder temporary storage and the RGBA conversion can coexist;
these budgets are not exact process-memory caps. Images outside the output still
consume decoding resources if their content must be interpreted.
Stencil accounting has two independent limits:

- Per flush: 8,388,608 pixels (33,554,432 RGBA bytes of requested regions), enough
  for one maximum-area draw and clip. A single draw and its distinct clip leaves
  are indivisible; exceeding this limit returns `ErrLimit`.
- Per plan: 67,108,864 pixels (268,435,456 RGBA bytes of cumulative region work),
  bounding rasterization work across all flushes.

Preparation marks batch boundaries on draws. The renderer flushes pending vector
fills there without clearing, resolving or resampling the color surface. Gradient
painting is explicitly finished as its own batch. Fill-rule transitions explicitly end pending
batches, including when the new draw is culled. Bounds are conservative,
intersected with draw/surface bounds and rounded to 16 pixels per axis. Clip
commands remain charged per submission.

These are region/work bounds, not physical GPU memory limits. Metal allocation
traces showed engine atlas pages growing to 16384×4096 when scratch targets shared
pages with vector's stencil textures. Scratch targets now use separate textures.
Vector's atlas rounding, row packing, engine page growth and retained clip masks
still amplify allocation: maximum-area single-flush gradient/clip probes reached
352.25 MiB of additional GPU image memory. This is an observed worst case among
the measured probes, not a universal physical bound. The rationale is beside the
constants in [renderplan.go](renderplan.go).
Complex but valid records can exceed these conservative internal limits.

## Text integration

`text.Draw`, `AppendGlyphs` and `AppendLazyGlyphs` use the same SVG path.
No changes to existing drawing calls are required. `GoTextFace.SVGForeground`
provides `currentColor` (nil means opaque black); `SVGPalette` supplies straight
RGBA palette entries (nil selects the first CPAL palette, an empty non-nil
slice supplies no entries). These inputs affect SVG only. COLRv0 and bitmap
color behavior is unchanged. This deliberately keeps color selection on the
face so public glyph-image access and Draw agree. RGB in Draw's ColorScale
continues to leave fixed SVG colors unchanged. ColorScale alpha applies to the
completed glyph once, separately from foreground/palette alpha. Palette alpha
is applied once during SVG paint interpretation. Lazy glyphs capture colors,
size, positioning and the small layout result before the face can change.

Advances, baselines and layout measurements remain font metrics. Image bounds
come from the supported artwork, not fallback outlines or the em square.
Preparation measures transformed curve extrema, conservative stroke envelopes,
clip unions/intersections and image viewports. These are conservative ink
bounds, not a minimal occupied-pixel rectangle: curved stroke joins, overlapping
clip envelopes and transparent image texels may leave extra space. No raster
readback is used. Negative origins and quantized fractional glyph positions
are included in both bounds and raster transforms. SVG candidates bypass the
font-metric origin heuristic; Draw's final transformed image-bounds cull still
runs before GPU allocation. `AppendVectorPath` continues to return outlines.

Document parsing, stylesheet parsing, interpretation, embedded-image decoding,
bounds and render preparation are CPU-only. A bounds pass and final surface
pass share their decoded embedded images; each pass enforces the interpreter's
geometry/reference/image limits, and the final pass charges the actual surface
work. Images are created only by glyph image realization in the graphics
lifecycle. Renderer scratch resources are released after realization. Draw's
existing realization-then-composition ordering is preserved.

The font source owns raw SVG records and a separate, mutex-protected SVG cache:

- Two parsed documents, including parsed stylesheets and parse/decode failures.
  Shared records with equal offset/length receive one document index. Parsing
  is independent of foreground and palette. Decompression occurs on document
  cache misses, never once per record in a shared-document cache hit.
- 512 lightweight bounds/error results. Layout hits do not parse, interpret or
  decode images, even when their heavy plans have been evicted. Colors remain
  part of this key because transparent foreground/palette paint can change ink.
- Four prepared glyph plans, including failures. Embedded-image storage belongs
  to a plan; there is no unbounded decoded-image cache.
- 128 final glyph images. Eviction drops references, as with other text images;
  images already returned to an application remain valid. Engine finalization
  releases unreachable images. No cache eviction deallocates a caller's image.

All four caches evict the least recently accessed entry at their hard limit,
even within one frame. Entries otherwise live until eviction or source release.
Lazy glyphs retained by an application independently retain only their small
layout result and immutable color key. A final-image cache miss obtains or
rebuilds its heavy plan; an image hit does not need the plan.
Document size/element budgets, per-plan command/image budgets and sample-surface
limits bound each entry. The text integration independently caps each final
image at 262,144 pixels, keeping image-cache retention bounded when the sampling
policy changes. This is a temporary safety cap, not a long-term supported-size
contract: oversized artwork falls back to its outline, which can be monochrome
or empty. Large-glyph compatibility and a byte-budgeted eviction policy remain
part of acceptance work. Engine allocation overhead is separate.

Plan/image keys include the source-owned document index, glyph ID, fixed font
size, quantized fractional origin and immutable foreground/palette bytes.
SVG geometry is independent of outline variation coordinates; shaping and
positioning still use the existing variation-aware caches. Sideways SVG remains
unsupported and uses the rotated fallback outline. Native-size vector AA is
the fixed rendering policy, so it is implicit in the key. Draw's final transform
and filter apply after the cached glyph is rendered and are not part of the
glyph key. Embedded images retain nearest filtering. Image-filter quality,
composition-edge handling and performance tuning remain acceptance work;
there is no size threshold or 2x/4x supersampling policy in normal text drawing.
Shader reuse across realizations is also deferred; scratch surfaces and shaders
are currently released together.
Nearest-image samples exactly on texel boundaries can change with GPU
interpolation rounding when a surface is translated/cropped. The allocation
comparison covers fractional positions away from these ties; stable tie
handling belongs to the remaining image-sampling evaluation.

### Selection, errors and fallback

The format priority remains COLRv0, bitmap, SVG and outline, with unsupported
COLRv1 using SVG before its outline. Existing COLRv0 layer filtering is preserved.
Malformed SVG tables are unavailable; malformed documents, unsupported rendered
features, missing glyph IDs, invalid references/images and resource limits reject
the entire SVG plan and use the existing outline fallback. No partial prefix is
rendered and the old oksvg path is never called. Valid empty artwork suppresses
fallback, including empty selection, disabled paint and empty clips. As before,
the public text drawing API does not return glyph-rendering errors. Internal
errors retain `errors.Is` identities (`ErrXML`, `ErrStyle`, `ErrGeometry`,
`ErrImage`, `ErrUnsupported`, `ErrGlyphNotFound`, reference errors and `ErrLimit`).
Shader/compiler failures after a valid plan are engine failures and panic rather
than masquerading as font fallback. Whole-document selection is allowed only
when there is no glyph ID anywhere; shared-document selection keeps the
interpreter's use-instance and ancestor semantics.

### Compressed loading boundary

The integration reads raw SVG records through the OpenType loader. It never
calls go-text's `GlyphData` or `GlyphDataSVG`, whose gzip decoder uses an
unlimited `io.ReadAll`. Instead it explicitly selects color, bitmap and outline
accessors and routes SVG through `Decode` before parsing. Font construction and
shaping do not decompress SVG records in go-text v0.3.5. No dependency change or
fork is needed. `UnsafeInternal` remains outside these guarantees.

Raw XML is limited to 32 MiB. Gzip input is limited to 8 MiB compressed and
32 MiB decoded, including concatenated gzip members. Standard-library gzip
validates the stream and trailer; invalid compression has `ErrCompression`
identity. A limiting reader permits at most one byte beyond the decoded budget
to distinguish overflow. This accommodates the roughly 14 MB shared Noto
document without trusting compressed length claims. These limits concern SVG
record processing, not a general font-loader or WOFF allocation budget.

## Validation

The parent package and `svgcss` tests remain CPU-only. Render preparation tests
cover errors, pruning, inheritance, analytic bounds, limits and fuzzing. GPU tests
live in `svgrender` with an external test package and a hidden-window run loop.
On Linux, invoke them under `xvfb-run`, including recursive package tests.

```sh
go test ./text/v2/internal/opentypesvg ./text/v2/internal/opentypesvg/svgcss
go test -race -count=1 ./text/v2/internal/opentypesvg ./text/v2/internal/opentypesvg/svgcss
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzDocument$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzPath$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzCoordinates$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzStyles$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzStyleValues$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzRenderPreparation$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg/svgcss -run '^$' -fuzz '^FuzzSyntax$' -fuzztime 10s
```

```sh
# macOS: the GPU TestMain hides its window.
go test -count=1 ./text/v2/internal/opentypesvg/svgrender
EBITENGINE_GRAPHICS_LIBRARY=opengl go test -count=1 ./text/v2/internal/opentypesvg/svgrender
# Linux:
xvfb-run -a go test -count=1 ./text/v2/internal/opentypesvg/svgrender
```

Deterministic tests cover rendering semantics, clip identity, full-sample
resolve, forced flush callbacks, composition across batches and selected
real-font pixels. All ten real records also prepare at 256 pixels with
8 samples per axis and at 512 pixels with 4 samples per axis. `TestIndependentReference` additionally
compares focused cases and four real-font records with `rsvg-convert` when it
is installed. Both sides use an explicit 8x density and full box averaging.
The checks require occupied-pixel mean channel error at most 5 and no pixel
with a channel error above 32. This is a bounded regression comparison, not
pixel equivalence or a production sampling policy.

librsvg 2.62.3/Cairo 1.18.4 disagrees on linearRGB gradients and percentage
coordinates in bounding-box clips. Those cases use analytic color and
normalized-coordinate equivalence tests. Set `EBITENGINE_SVG_RENDER_REPORT_DIR`
to an existing directory to retain comparison PNGs.

[Composition tests](../../../../vector/clipgroup_test.go) provide the separate
baseline for nested isolated opacity. The `group-identity`,
`nested-groups`, `embedded-png` and generated `embedded-jpeg` fixtures render.
Composition tests compare native point-sample oracles at densities 1, 4 and 8
and offsets 0 and 0.375. Image mapping tests compare against an inverse affine
texel-selection oracle with binary viewport/clip membership. Independent
comparisons also cover nested opacity, overlapping fill/stroke and the JPEG
fixture; their existing error thresholds are regression bounds. Full-font evaluation,
additional platforms and production quality/performance acceptance remain necessary
before removing the unused oksvg implementation and its dependencies.

[opentype]: https://learn.microsoft.com/en-us/typography/opentype/spec/svg
[svg-use]: https://www.w3.org/TR/SVG11/struct.html#UseElement
