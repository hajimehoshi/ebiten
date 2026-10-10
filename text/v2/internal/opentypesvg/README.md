# OpenType SVG

CPU-only foundations for the internal glyph renderer planned in
[#3796](https://github.com/hajimehoshi/ebiten/issues/3796). Document parsing,
glyph selection, retained geometry, viewport mapping, styles, colors and
linear/radial gradients are implemented. Rendering and text integration are
not implemented. Successful interpretation does not establish complete
rendering support.

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
| Composition inputs | Fill/clip rules, stroke parameters, clip references, opacity, display and visibility |

Geometry stays as curves until output scale is known. Units and transforms use
explicit contexts; source ancestry is not an instance transform parent. The
root viewport preserves font coordinates without introducing an implicit clip.
The generic geometry helper supports em/ex with supplied metrics; style and
gradient lengths enforce the OpenType restriction on those units.

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
OpenType element subtrees are ignored by the document parser. Other element
support still needs interpreter-level classification.

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

## Renderer handoff

The later interpreter must:

- Supply use-instance parents and compose local, viewport and output transforms
  explicitly. Do not apply a selected root's style or transform twice.
- Sample paints with their separate paint opacity, then apply element/group
  opacity to the composed result. Premultiply only at the rendering boundary.
- Provide object bounds and viewport lengths for gradients; honor disabled and
  solid-color results, color interpolation, spread modes and focal behavior.
  Singular transforms and a focus on the circle need sampling treatment.
- Expand/dash strokes, traverse and bound clip references, perform clipping and
  composition, and prune display:none subtrees while retaining definitions.
  Hidden visibility can be overridden by descendants.
- Decide how to report or draw completed geometry prefixes on malformed input.
  Bound aggregate geometry and any additional instance/reference expansion.
- Bound gzip decompression before document parsing. go-text currently
  decompresses before this package receives the source. Check embedded image
  dimensions against a pixel budget before decoding PNG/JPEG data.

Unresolved or wrong-type references return `errReference`; cycles return
`errReferenceCycle`. Explicit paint fallbacks can handle unusable gradients.
`errLimit` and `errUnsupported` always propagate. Geometry, paint sampling,
embedded images, text integration and caching still need rendering validation.

## Limits

Inputs are bounded: decoded documents allow 32 MiB, CSS text 1 MiB, and
individual SVG property/attribute values 16 KiB. The CSS parser independently
limits value text to 16 KiB; ParseValue excludes priority and surrounding
whitespace from that limit. Raw declaration input remains bounded by 1 MiB.

Depth, element, rule, selector, stop and dash limits are declared beside their
parsers in [document.go](document.go), [number.go](number.go),
[css.go](css.go) and [svgcss/syntax.go](svgcss/syntax.go). Resolver work,
instances and cached values have independent budgets; expensive input can hit
`errLimit` before another ceiling is reached. Work charging includes repeated
parsing of inherited color and stroke values. Cache-entry checks are defensive
backstops to the document element limit; the shared cached-value budget is a
reachable limit. These are record/work bounds, not allocator byte guarantees.

## Validation

All package tests are CPU-only and require no display or graphics harness.
They cover fixtures, malformed input, cascade, instance inheritance, paint
interpretation and resource limits. They do not establish pixel compatibility.

```sh
go test ./text/v2/internal/opentypesvg/...
go test -race -count=1 ./text/v2/internal/opentypesvg/...
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzDocument$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzPath$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzCoordinates$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzStyles$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg -run '^$' -fuzz '^FuzzStyleValues$' -fuzztime 10s
go test ./text/v2/internal/opentypesvg/svgcss -run '^$' -fuzz '^FuzzSyntax$' -fuzztime 10s
```

[Composition tests](../../../../vector/clipgroup_test.go) provide a separate
baseline for nested opacity and clipping. Run graphics tests under xvfb-run on
Linux. Raster quality, JPEG coverage, full-font evaluation and independent
comparisons at multiple sizes and fractional offsets remain acceptance work
before replacing oksvg.

[opentype]: https://learn.microsoft.com/en-us/typography/opentype/spec/svg
[svg-use]: https://www.w3.org/TR/SVG11/struct.html#UseElement
