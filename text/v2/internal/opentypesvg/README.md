# OpenType SVG

CPU-only document foundation for the planned internal glyph renderer
([#3796](https://github.com/hajimehoshi/ebiten/issues/3796)), with
[acceptance data](testdata/README.md) for the later rendering stages.

The document model, local fragment lookup and glyph selection with bounded
`use` href traversal are implemented. Geometry, CSS evaluation, paint,
rendering and text integration remain future work. CSS and paint-reference
traversal, including gradient and pattern href chains, must be bounded after
effective properties are determined. Instance inheritance and CSS selector
ancestry need interpreter-level tests. Embedded image decoding must check
`image.DecodeConfig` dimensions against a pixel budget before `Decode`.
go-text decompresses gzip-encoded SVG documents without a size limit before
`parse` sees them, so the decoded byte limit does not protect against a
compression bomb; integration must bound decompression or avoid that path.
The document model and its operations are private to this package. External
package tests access them through aliases and wrappers in `export_test.go`.
Implementation constraints are documented in `document.go`.

Run the CPU-only tests without a display or graphics harness:

```sh
go test ./text/v2/internal/opentypesvg
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz FuzzDocument -fuzztime 10s
```

## Supported profile

Target the required capabilities of [OpenType SVG 1.9.1](https://learn.microsoft.com/en-us/typography/opentype/spec/svg)
and its referenced SVG 1.1 specification. This checklist describes intended support.
Fixture names refer to `testdata/`.

| Status | Requirement | Acceptance input |
| --- | --- | --- |
| Required | UTF-8 XML, plain and gzip documents | `shared-context.svg` and `.svg.gz` |
| Required | Glyph IDs, shared definitions, local references | Noto 2–3; shared-context |
| Required | Paths, shapes, transforms, viewport mapping | Both fonts; stroke-transform; viewport-overflow |
| Required | Fill/stroke properties, inheritance and fill rules | evenodd-clip; stroke-transform |
| Required | Linear/radial gradients and references | Noto 3330/3629; gradient-reference |
| Required | Clipping and isolated opacity | Twemoji 1039/1355; group-identity; nested-groups |
| Required | Named/numeric colors and currentColor | CSS fixture; palette |
| Required | Embedded PNG/JPEG | embedded-png; JPEG fixture still needed |
| Optional, preserve | Inline styles and simple CSS class rules | CSS fixture; existing oksvg tests |
| Optional, planned | CPAL variables and color fallbacks | palette |
| Optional, deferred | Filters, patterns, masks, markers, symbols, animation, nested SVG, entities, interactivity | No acceptance claim |
| Restricted | Ignore prohibited elements; no scripts or external resources | restricted |


Use the [oksvg tests](../oksvg/oksvg_test.go) and
[glyph-document tests](../../gotextsvgdoc_test.go) as compatibility references.
The first retained element in document order wins for each duplicate ID.
Malformed XML is rejected, and an unresolved structural reference fails that
glyph's selection. A document without the requested `glyphN` returns
`errGlyphNotFound`. Integration may render the whole document using `rootElement()`
only when no `glyphN` element exists anywhere in the document and the root
carries no glyph ID. When other `glyphN` elements exist, the missing glyph has
no SVG artwork, and text rendering falls back to its outline glyph, as the
current `text/v2` integration does.

CSS class declaration order must win independently of class-token order;
repeated classes must not compound opacity. Use specifications and independent
renderers as references, not oksvg output.

Remaining rendering acceptance work includes JPEG, units, gradient spread/focal
behavior, stroke joins/dashes and image aspect ratio. Compare real glyphs
independently at 16, 24, 64 and 256 pixels with offsets 0 and 0.375 before
replacing oksvg.

## Composition baseline

[vector/clipgroup_test.go](../../../../vector/clipgroup_test.go) checks nested
half-opacity groups with identical or overlapping colored children and circle/
rectangle clips. Its 32 configurations cover 1x, 2x, 4x and 16x sampling, offsets
0 and 0.375, and clipping at group or child level. Point samples survive all
composition steps before a full box resolve; every premultiplied RGBA channel
must match the per-sample oracle within one byte.

`PathClip` and `ClipSet` suffice for these invariants. Rasterization quality,
sampling density, performance and caching remain open. Full-font evaluation is
also needed: Noto has a shared SVG document of about 14 MB, omitted from the corpus.

Run from the repository root (macOS uses the hidden-window test harness):

```sh
go test ./vector -run 'TestClip(NestedPointSampledGroups|PointSampledOpacityGroup|Identity|Composition)$'
EBITENGINE_GRAPHICS_LIBRARY=opengl go test ./vector -run TestClipNestedPointSampledGroups
```

On Linux, prefix graphics tests with `xvfb-run -a`.
The full vector suite and focused OpenGL checks passed on macOS on 2026-10-05.
Other platforms and independent real-font rasterization comparisons remain unverified.
