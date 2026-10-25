# SVG acceptance data

Ten unmodified SVG table records and twelve focused SVG fixtures for the
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
| shared-context | Glyph 1 occupies [0,8)×[0,8); glyph 2 [16,24)×[0,8), both with initial black fill. Render only the selected use instance and its shared path; the original ancestor transform/fill does not apply. Gzip and plain text agree; missing glyph 3 produces no artwork. |
| stroke-transform | First line has four-unit output stroke width with round caps, centered at y=16 from x=12 to x=36. (24,16) is green; (24,20) is transparent. Curves keep their original commands until output scale is known. |
| palette | With foreground blue and no palette, (8,8) is blue and (24,8) red. With palette entry 0 RGB(0,255,0) at alpha 0.5, (24,8) is (0,128,0,128). Palette alpha must enter paint opacity once. |
| embedded-jpeg | Original 8×8 solid RGB(80,160,240) image encoded using Go image/jpeg at quality 100, mapped to [4,36)×[4,36). Interior RGB is within one byte of the source and alpha is 255. |
| embedded-png | Red left half, blue right half of [4,36)×[4,20); (8,12) red and (30,12) blue, both opaque. No network access. |
| restricted | Only the red [0,8)×[0,8) tile appears. Prohibited element subtrees do not render or execute. |
| viewport-overflow | The entire red rectangle [-4,12)×[-8,8) survives despite root overflow=hidden. At an output origin shifted to (16,16), (13,10) is red. Root viewport is not an implicit glyph clip. |


## CPU style/paint interpretation

The style and gradient tests now interpret the real records without graphics
initialization and verify focused cascade, palette and gradient expectations.
These checks do not establish the pixel expectations above. `style_test.go`
also tests use-instance inheritance separately from source selector ancestry,
including the SVG 1.1 style-cloning behavior. Gradient tests supply independent
viewport and object bounds and check definitions, transforms, inherited stops,
stop alpha and references without flattening geometry or sampling pixels.
The resolver's supported subset, compatibility allowances, limits and error
policy are documented in [the package README](../README.md#supported-behavior).

## GPU validation

`../svgrender` exercises the focused vector fixtures and selected real-font
records through `PrepareRender`. Tests keep a point-sample grid until the final
GPU box resolve. Group opacity and both embedded-image fixtures are supported.
See [renderer validation](../README.md#validation) for test commands,
independent-comparison scope and limitations.

The JPEG fixture is generated original test data under Apache-2.0, with no
third-party source. To reproduce its payload, fill an 8×8 image.RGBA with
color.RGBA{R: 80, G: 160, B: 240, A: 255}, encode with the standard-library
image/jpeg encoder and jpeg.Options{Quality: 100}, and base64-encode the bytes.

## Full-font preparation inventory and bounded smoke checks

This is an incomplete acceptance workflow. It does not establish full-font GPU
compatibility, pixel correctness, performance acceptance, or platform acceptance.

`TestSVGFontInventory` defaults to SVG-record preparation only. It inventories
all unique records and glyph selections at 24 and 96 pixels, using the text
cache and the horizontal origin quantization of normal layout. It deliberately
prepares SVG candidates independently of color-format priority. These counts
must not be described as successful production format selection.

Use the fixed licensed font versions listed above. The report
records the font SHA-256 and each encoded document's SHA-256, offset, length,
decoded size, and glyph ranges. Verify the input hash when comparing runs.
Noto 2.051 used during development had SHA-256
`7fb39738ab18f10612d6f4595e2e8e47a0afdf34738460442d99cd0c344a4d90`.
No font binaries or generated reports belong in the repository.

From the repository root, with an existing output directory:

```sh
EBITENGINE_SVG_ACCEPTANCE_FONT=/path/font.ttf EBITENGINE_SVG_ACCEPTANCE_REPORT_DIR=/path/report \
  go test -p 1 -count=1 -run '^TestSVGFontInventory$' -v ./text/v2
```

This test still uses text/v2's hidden graphics test lifecycle, even when it
only prepares records; it is not a display-free parser test. On Linux prefix
with `xvfb-run -a`. Ordinary parser/style tests remain display-free.

`full-font.json` separates malformed, unsupported, resource-limited, valid-empty,
and successfully prepared candidates. `Phase` identifies preparation;
`Realized` identifies rows whose realization completed and was read back;
those pixels are not validated.
`Complete` means the inventory loop completed, not acceptance. Decode failures
are recorded and their glyph selections are still classified. A deferred report
write preserves progress on ordinary test failures, but cannot survive process
termination. Use a separate output directory for each run.

GPU work is off by default. `EBITENGINE_SVG_ACCEPTANCE_GPU_GLYPHS=1` through `8` enables
only that many realizations from the inventory, skipping images larger than
256 pixels on either axis. This is a smoke sample, not a full-font GPU sweep.
Optionally set `EBITENGINE_SVG_ACCEPTANCE_TEXT` to text shaping into at most eight glyphs
with images no larger than 256 pixels per axis. That checks SVG cache identity
and colored status through public glyph access and compares `text.Draw` with
explicit glyph composition. Choose nonempty SVG glyphs, without preferred bitmap
or COLRv0 data; other selections fail this SVG-specific check. It is not an
independent rasterization oracle. It does not cover every ligature.

Run a single process at a time, without VMs or other acceptance runs, with
`-p 1`. Counts and image dimensions bound submitted work, not physical GPU memory:
engine atlases, scratch surfaces and finalizer timing still add overhead. Do not
expand the sample into a full sweep before measuring isolated peak memory.

### Benchmarks

Benchmarks are separately opt-in and limited to at most 16 iterations per case:

```sh
EBITENGINE_SVG_ACCEPTANCE_BENCHMARK=1 go test -p 1 -run '^$' \
  -bench '^BenchmarkSVGProduction/warm$' -benchtime=16x ./text/v2
```

On Linux, prefix the benchmark command with `xvfb-run -a`.
Run one named workload per process. `cold` includes synthetic font construction
and source loading, so it is not a pure SVG rasterization benchmark. `warm`
is prewarmed; `layout` measures cached lazy layout. `colors` changes currentColor
and misses the image cache. `large` uses a 512-pixel em to stay within the current
image cap. Every drawing workload rejects monochrome outline fallback. Readback
synchronizes each draw, so times measure latency, not asynchronous throughput.
Shader compilation on image misses is included. These short bounded probes are
not statistically robust performance acceptance measurements.
