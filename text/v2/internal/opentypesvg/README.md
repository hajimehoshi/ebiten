# OpenType SVG

Internal OpenType SVG glyph interpretation and GPU rendering for `text/v2`
([#3796](https://github.com/hajimehoshi/ebiten/issues/3796)). Parsing, styles,
geometry, image decoding and render preparation are CPU-only;
`svgrender.Renderer` draws the prepared plan within Ebitengine's graphics
lifecycle. See [testdata](testdata/README.md) for fixtures, licenses and
full-font acceptance procedures.

## Supported behavior

The target is the required subset of [OpenType SVG][opentype] and SVG 1.1,
not general browser rendering.

| Area | Supported |
| --- | --- |
| Documents | UTF-8 XML, glyph selection, shared definitions and local references |
| Geometry | Paths, basic shapes, transforms, lengths and viewport mapping |
| Styles | Presentation attributes, inline styles, CSS cascade and inheritance |
| Paint | SVG colors, currentColor, CPAL variables, local URL paints and fallbacks |
| Gradients | Linear/radial gradients, href inheritance, spread and transforms |
| Rendering | Fills, strokes, dashes, nested clips and isolated element/group opacity |
| Images | Embedded PNG/JPEG data URIs, transforms, viewport mapping and clips |

CSS supports ASCII type, universal, ID and class selectors, compounds,
descendant/child combinators and selector lists. Attribute, pseudo, sibling
and namespace selectors, escaped/non-ASCII identifiers, at-rules and
custom-property definitions are unsupported. Unknown properties are ignored;
invalid declarations are discarded before the cascade. Malformed stylesheets
fail. [css.go](css.go) and [style.go](style.go) define the property vocabulary
and applicability rules; [svgcss](svgcss) handles syntax only.

Filters, patterns, effective masks, markers, CSS transforms/geometry, nested
SVG viewports, symbol instances and pathLength-calibrated dashes are unsupported.
Effective unsupported features reject the plan; overridden, unmatched or pruned
features do not. Restricted OpenType and animation subtrees are ignored.
External URLs and local files are never loaded.

Selectors follow source ancestry and inheritance follows the instance parent,
using [SVG 1.1 use semantics][svg-use]. Whole-document selection is allowed only
when no glyph ID exists anywhere; duplicate IDs select the first retained
element. The root viewport does not implicitly clip artwork. Clip content is
limited to shapes and uses that directly reference shapes.

## Rendering

`PrepareRender(source, RenderOptions)` takes a glyph ID, UPEM, output dimensions,
font-to-output transform, foreground/palette and sample density (1, 2, 4 or 8
per axis). Foreground is straight `color.NRGBA`; opaque black is the ordinary
default. A zero transform means identity. Treat the returned plan as immutable.
Reuse a `svgrender.Renderer` and call its `Draw` and `Deallocate` methods within
the graphics lifecycle. Draw releases scratch images; Deallocate releases
retained shaders.

Normal text drawing uses `Samples: 1, AntiAlias: true` at native glyph resolution,
with no supersampling or downsampling. A path and its clip share coverage, but
separate draws resolve coverage before composition. Coincident paints can leave
fringes and adjoining fills can leave seams, including inside opacity groups.
The internal point-sample mode preserves samples through composition for
reference tests.

Embedded images use premultiplied area prefiltering for minification and clamped
bilinear sampling. SVG image-rendering modes are unsupported; general affine
sampling is not an exact pixel-area or anisotropic filter. Composition edges,
affine image quality and full-font visual/performance acceptance remain open.

Preparation errors reject the entire plan, with no partial artwork. Use
`errors.Is` with the exported error identities to distinguish malformed input,
unsupported features, missing glyphs, invalid references and resource limits.

## Limits

Input, traversal, geometry, image decoding and rendering work have independent
budgets. Complex valid artwork can exceed them and return `ErrLimit`.

| Resource | Limit |
| --- | --- |
| SVG record | 32 MiB decoded; 8 MiB compressed for gzip |
| CSS | 1 MiB text; 16 KiB per property/attribute value |
| Output surface | 4096 pixels per axis; 4,194,304 pixels total |
| Opacity groups | 16 simultaneous levels; 16,777,216 group sample pixels |
| Embedded images | 256 distinct URIs; 4 MiB per URI; 16 MiB total URI bytes |
| Decoded images | 4096 pixels per axis; 4,194,304 pixels each; 8,388,608 total |

Exact budgets and accounting are defined beside the parsers and in
[renderplan.go](renderplan.go). These bound records, requested pixel regions
and work, **not physical process or GPU memory**. Decoder storage, atlas
allocation, driver caches and application-held images can add memory beyond
these budgets. Gzip limits include concatenated members; SVG record limits do
not establish a general font-loader or WOFF allocation budget.

## Text integration

`text.Draw`, `AppendGlyphs` and `AppendLazyGlyphs` share this renderer.
SVG currentColor is opaque black, and palette entries come from the font's
first CPAL palette. Draw's ColorScale RGB leaves fixed SVG colors unchanged;
its alpha applies once to the completed glyph. Lazy glyphs capture layout
before the face changes.

Layout uses font metrics; image bounds use conservative artwork bounds,
including negative origins. `AppendVectorPath` continues to return outlines.
Format priority is COLRv0, bitmap, SVG, then outline. Unsupported COLRv1 can
use SVG; sideways SVG uses the rotated outline. Invalid or unsupported SVG and
resource-limit failures fall back to outlines. Valid empty artwork suppresses
fallback. Public drawing APIs do not return glyph-rendering errors; shader or
compiler failures panic.

Each font source owns bounded LRU caches: two parsed documents, 512 bounds/error
results, four prepared plans, and up to 128 final images totaling 32 MiB of
requested RGBA pixels. Eviction never deallocates images returned to callers
or guarantees immediate GPU reclamation. SVG loading uses bounded `Decode`
instead of go-text's unbounded SVG decompression path; `UnsafeInternal` is
outside these guarantees.

## Validation

Run CPU tests from the repository root:

```sh
go test ./text/v2/internal/opentypesvg ./text/v2/internal/opentypesvg/svgcss
go test -race -count=1 ./text/v2/internal/opentypesvg ./text/v2/internal/opentypesvg/svgcss
```

GPU tests use a hidden-window run loop. On Linux, use `xvfb-run`, including for
recursive package tests:

```sh
# macOS
go test -count=1 ./text/v2/internal/opentypesvg/svgrender
EBITENGINE_GRAPHICS_LIBRARY=opengl go test -count=1 ./text/v2/internal/opentypesvg/svgrender
# Linux
xvfb-run -a go test -count=1 ./text/v2/internal/opentypesvg/svgrender
```

Tests cover interpretation, bounds, limits, composition and selected real-font
pixels. `TestIndependentReference` also compares with `rsvg-convert` when
installed. Set `EBITENGINE_SVG_RENDER_REPORT_DIR` to an existing directory to
retain comparison PNGs. These are regression checks, not full-font visual or
performance acceptance. See the [fixture guide](testdata/README.md) for
production-reference checks, bounded inventory runs and benchmarks.

[opentype]: https://learn.microsoft.com/en-us/typography/opentype/spec/svg
[svg-use]: https://www.w3.org/TR/SVG11/struct.html#UseElement
