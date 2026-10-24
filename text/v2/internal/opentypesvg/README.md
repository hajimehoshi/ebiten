# OpenType SVG

Internal glyph interpretation and GPU rendering for
[#3796](https://github.com/hajimehoshi/ebiten/issues/3796). Document parsing,
glyph selection, retained geometry, viewport mapping, styles and paint
interpretation remain CPU-only. `PrepareRender` produces a bounded render plan;
`svgrender.Renderer` uses vector paths and `vector.ImagePaint` for gradient paints. Text integration and glyph caching are not implemented. Successful
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
| Composition | Paint opacity, display/visibility and single-paint element opacity; isolated groups deferred |

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
`symbol` instances, pathLength-calibrated dashes and embedded images are
currently unsupported.

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
on the chosen sample grid and applied through `vector.ImagePaint`; final resolve
still occurs only after composition. Sampling density remains an explicit
evaluation choice; no production quality policy has been selected.

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

Isolated element/group opacity and embedded PNG/JPEG rendering remain item 6.
Zero opacity produces no draws and skips painting effects. Such a subtree is
otherwise skipped unless an active ancestor needs its geometry for an
objectBoundingBox clip. In that case its geometry, transforms, display values
and use references contribute to the bounds; errors needed to resolve those
bounds propagate. Filters, masks and other painting effects remain ignored.
Opacity is ignored in clip content.
Opacity strictly between zero and one on containers or elements with both fill
and stroke returns `ErrUnsupported`. Single-paint leaf opacity and independent
fill/stroke opacity are supported. Text integration, caching, external foreground/palette
plumbing, gzip limits, production quality selection and replacement of oksvg
remain separate work. The sample-preserving composition requirements for item 6
are documented beside the surfaces and final resolve in
[renderer.go](svgrender/renderer.go).

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
coordinates and sample surfaces. Stencil accounting has two independent limits:

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
normalized-coordinate equivalence tests. Set `SVG_RENDER_REPORT_DIR`
to an existing directory to retain comparison PNGs.

[Composition tests](../../../../vector/clipgroup_test.go) provide the separate
baseline for nested isolated opacity. The renderer rejects the `group-identity`,
`nested-groups` and embedded-image fixtures until item 6. Full-font evaluation,
additional platforms and caching/integration acceptance remain necessary before
replacing oksvg.

[opentype]: https://learn.microsoft.com/en-us/typography/opentype/spec/svg
[svg-use]: https://www.w3.org/TR/SVG11/struct.html#UseElement
