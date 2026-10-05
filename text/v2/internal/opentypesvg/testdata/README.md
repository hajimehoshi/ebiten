# SVG acceptance data

Ten unmodified SVG table records and eleven focused SVG fixtures for the
[planned renderer](../README.md). License texts and attribution are in
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).

## Sources and regeneration

- [Noto Color Emoji 2.051](https://res.ebitengine.org/examples/NotoColorEmoji-Regular.ttf),
  name-table version `Version 2.051;GOOG;noto-emoji:20250818:e92753bfa55fd449e427d4d325f9c8c40408c74e`.
- [Twitter Color Emoji SVGinOT 15.1.0](https://github.com/13rac1/twemoji-color-font/releases/download/v15.1.0/TwitterColorEmoji-SVGinOT-15.1.0.zip):
  extract `TwitterColorEmoji-SVGinOT-15.1.0/TwitterColorEmoji-SVGinOT.ttf`.

The tool generates `inventory.json` with UPEM, glyph ranges and complete-font
construct counts for inspection; this file is not checked in. Counts are per
unique SVG document; attributes use local names (`href` includes `xlink:href`).
Noto contributes gradients, quadratic paths and shared `use` references; Twemoji
adds cubic/arc paths, basic shapes, clipping and even-odd rules.

Run from this directory using Go's standard library:

```sh
go run extract.go /path/NotoColorEmoji-Regular.ttf /path/TwitterColorEmoji-SVGinOT.ttf
```

Use the font versions above. The tool overwrites `real/*.svg` and generates
`inventory.json`. Compare the SVG output with the checked-in fixtures. Keep
downloaded fonts for offline regeneration; the Noto URL can change. Licenses
are maintained separately.
The gzip fixture must decompress to the identical `shared-context.svg` bytes.

## Expectations

Focused fixtures use viewport/UPEM 48, glyph 1, one pixel per user unit and transparent
black unless specified below. Colors are premultiplied RGBA bytes, with one byte
of rounding tolerance. Apply offsets to shape and clip together; edge expectations
express composition invariants rather than fixed rasterization goldens.

### Real font records

| Record | Expected behavior |
| --- | --- |
| Noto 2–3 | Select only the requested keycap tile. Glyph 2 uses dark gray; glyph 3 light gray. Both reference shared paths. At font point (650,-350), fills are #424242 and #E0E0E0 respectively. |
| Noto 3330 | Render a shaded numeral 1 from quadratic paths and a user-space linear gradient; extend the final stop beyond offset 0.515. |
| Noto 3629 | Render the layered burst with a yellow user-space radial gradient, preserving painter order and negative-y coordinates. |
| Noto 3682 | Render the flame with two transformed radial gradients; preserve the translucent inner highlight and its zero-alpha final stop. |
| Noto 3737 | Render a red octagon, darker lower/right edge, and a 0.9-opacity pale highlight. Do not clip the artwork at the em square. |
| Twemoji 692 | Render the red pin head, gray needle, and dark ellipse with the nested translation/scale intact. |
| Twemoji 839 | Render the sunglasses path with even-odd fill; preserve its explicit clip-rule for use as clip geometry. |
| Twemoji 1039 | Apply the referenced square clip to its group and retain the separate white detail in painter order. |
| Twemoji 1338 | Render the layered artwork with twelve user-space clips, each transformed with its referencing path. Clip coordinates must follow each local matrix. |
| Twemoji 1355 | Render a window with pale blue panes, a half-opacity white triangular reflection, and opaque brown crossbars above it. |


### Focused fixtures

| Fixture | Explicit expected behavior |
| --- | --- |
| css-classes | Pixels (4,4), (12,4), (20,4) are (0,0,128,128); (30,4) is transparent. Keep these results after selecting glyph 1, including stylesheet dependencies. |
| group-identity | Same result as one unclipped blue circle at opacity 0.5, including identical clip/shape edges. Center (24,24) is (0,0,128,128), never alpha 191. |
| nested-groups | Red circle then blue circle inside two half-opacity groups, clipped to the first circle and x<30. (10,24) is (64,0,0,64), (24,24) is (0,0,64,64), (35,24) transparent. Preserve the shared edges through both groups. |
| evenodd-clip | (8,8) is opaque red; (24,24) and (0,0) transparent. The inner contour is a hole despite matching winding. |
| gradient-reference | Top rectangle interpolates red to blue through inherited stops. Bottom uses the later radial definition; near (24,32) it is bright and opaque, darkening and losing opacity toward its radius. Forward and backward definition order must agree. |
| shared-context | With foreground RGB(0,255,0), glyph 1 occupies [8,16)×[8,16); glyph 2 [24,32)×[8,16). Render only the selected glyph while preserving the ancestor transform/fill and shared path. Gzip and plain text agree; missing glyph 3 produces no artwork. |
| stroke-transform | First line has four-unit output stroke width with round caps, centered at y=16 from x=12 to x=36. (24,16) is green; (24,20) is transparent. Curves keep their original commands until output scale is known. |
| palette | With foreground blue and no palette, (8,8) is blue and (24,8) red. With palette entry 0 RGB(0,255,0) at alpha 0.5, (24,8) is (0,128,0,128). Palette alpha must enter paint opacity once. |
| embedded-png | Red left half, blue right half of [4,36)×[4,20); (8,12) red and (30,12) blue, both opaque. No network access. |
| restricted | Only the red [0,8)×[0,8) tile appears. Prohibited element subtrees do not render or execute. |
| viewport-overflow | The entire red rectangle [-4,12)×[-8,8) survives despite root overflow=hidden. At an output origin shifted to (16,16), (13,10) is red. Root viewport is not an implicit glyph clip. |
