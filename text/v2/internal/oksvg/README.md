# oksvg (fork)

This package is a fork of [github.com/srwiley/oksvg](https://github.com/srwiley/oksvg)
at version `v0.0.0-20221011165216-be6e8873101c`. The original is licensed
under the BSD-3-Clause license; its full text is reproduced in the License
section of the repository root README.md. This fork is distributed under
the Apache License 2.0 like the rest of this repository.

The tests, testdata, and doc directory of the original repository are not
included.

## Local modifications

- Gradient rendering under a drawing transform: `SvgPath.DrawTransformed`
  now builds gradient color functions with
  `Gradient.GetColorFunctionUS(opacity, objMatrix)`, passing the composed
  user-to-device matrix, instead of `Gradient.GetColorFunction(opacity)`,
  which ignores the transform. Without this, gradients with
  `gradientUnits="userSpaceOnUse"` were evaluated against untransformed
  geometry and rendered incorrectly when a transform was applied.
- The `golang.org/x/net/html/charset` dependency is removed:
  `ReadIconStream` no longer sets `decoder.CharsetReader`. Consumers of
  this fork feed it OpenType SVG documents, which are UTF-8 by
  specification, and encoding/xml handles UTF-8 natively.
- Mechanical modernizations applied by `go fix` (e.g. `interface{}` to
  `any`).
- The original license headers are replaced with SPDX-style headers
  crediting the original authors.

## Missing features for OpenType SVG

This package implements a subset of SVG 1.1 and does not yet cover
everything that the [OpenType SVG table
specification](https://learn.microsoft.com/en-us/typography/opentype/spec/svg)
allows in glyph documents. The SVG documents in current Noto Color Emoji
fonts stay within the supported subset, but other fonts may not. Note that
parsing with `IgnoreErrorMode` skips unsupported constructs silently, so a
glyph using them renders with missing shapes or wrong colors rather than
failing.

Required by the OpenType SVG specification but not implemented:

- `<clipPath>` elements and the `clip-path` property. The specification
  names clipping paths as must-support alongside gradients and opacity.
- `currentColor`. The specification requires it, initialized to the text
  foreground color. `ReadReplacingCurrentColor` substitutes it textually
  before parsing, but the text/v2 caller uses `ReadIconStream`, and
  gradient stops do not inherit it.
- `<image>` elements with embedded base64 PNG or JPEG data.
- The `fill-rule` property. `evenodd` is not parsed; all paths are filled
  with the nonzero rule.
- Gradient templating via `xlink:href` on `<linearGradient>` and
  `<radialGradient>` (a gradient inheriting stops and attributes from
  another). `href` is resolved only on `<use>`.
- `preserveAspectRatio`, and percentage units in some places.

Optional in the specification but strongly recommended, not implemented:

- The CSS `var()` function with a fallback value, and CPAL palette colors
  referenced as `--color<N>` custom properties. A color value using
  `var()` fails to parse and is dropped instead of applying the fallback.
- Inline `style` attributes on gradient `<stop>` elements
  (`style="stop-color:..."`). Inline styles are parsed on shape and group
  elements, but stops read only the `offset`, `stop-color`, and
  `stop-opacity` presentation attributes.

Optional in the specification and not implemented. These are essentially
absent from real emoji fonts: the specification tells font authors to avoid
them for interoperability, and Google's font build pipeline
(picosvg/nanoemoji) normalizes glyph documents down to paths, groups,
gradients, and presentation attributes. The implementation cost varies:

- `<filter>`: a raster-effects pipeline (blur, compositing, lighting)
  requiring off-screen buffers; by far the most expensive to implement.
- `<mask>`: requires rendering to an off-screen alpha buffer and
  compositing; rasterx draws scanlines directly to the target, so this is
  an architectural change, not a parser addition.
- `<pattern>`: requires rendering a tile and exposing it as a paint
  source, similar plumbing to gradients but with an image lookup.
- `<marker>`: arrowheads on stroke vertices, aimed at diagrams; no known
  use in fonts.
- `<symbol>`: roughly `<defs>` plus `<use>` with its own viewport; easy,
  but never needed because emoji pipelines inline everything.
- CSS stylesheets: correct support means selector matching, specificity,
  and the cascade; only trivial single-class rules are handled. The
  specification recommends presentation attributes instead.
- Animations: the specification defines a static rendering as ignoring
  animations, so skipping the animation elements is the conformant static
  behavior.
- XML entities: Go's `encoding/xml` does not expand custom DTD entities,
  which is also the safe default (entity expansion is a classic attack
  vector).
- Interactivity: meaningless in a glyph rasterizer; there is no DOM or
  event model.

Elements banned by the specification (`<text>`, `<foreignObject>`,
`<switch>`, `<script>`, `<a>`, `<view>`) must be ignored by renderers; the
parser's unknown-element skipping already handles them.
