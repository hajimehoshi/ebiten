# OpenType SVG

Compatibility baseline for the planned internal glyph renderer
([#3796](https://github.com/hajimehoshi/ebiten/issues/3796)). This directory
currently contains [acceptance data](testdata/README.md); implementation is pending.

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
| Required | Named/numeric colors and currentColor | CSS fixture; shared-context; palette |
| Required | Embedded PNG/JPEG | embedded-png; JPEG fixture still needed |
| Optional, preserve | Inline styles and simple CSS class rules | CSS fixture; existing oksvg tests |
| Optional, planned | CPAL variables and color fallbacks | palette |
| Optional, deferred | Filters, patterns, masks, markers, symbols, animation, nested SVG, entities, interactivity | No acceptance claim |
| Restricted | Ignore prohibited elements; no scripts or external resources | restricted |


Preserve useful behavior covered by the [oksvg tests](../oksvg/oksvg_test.go)
and [glyph-document tests](../../gotextsvgdoc_test.go), including CSS classes,
transforms and shared definitions. Class declaration order must win independently
of class-token order; repeated classes must not compound opacity. Preserve CSS
and ancestor context during glyph selection. Use the specification and independent
renderers as references, not oksvg output.

Remaining acceptance work includes JPEG, namespace/unit handling, gradient
spread/focal behavior, stroke joins/dashes, image aspect ratio, invalid input and
bounded reference expansion. Compare real glyphs independently at 16, 24, 64 and
256 pixels with offsets 0 and 0.375 before replacing oksvg.

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
