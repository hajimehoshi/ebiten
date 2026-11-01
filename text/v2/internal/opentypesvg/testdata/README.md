# SVG acceptance data

Ten unmodified SVG table records and twelve focused SVG fixtures for the
[planned renderer](../README.md). License texts and attribution are in
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).

## Sources and regeneration

- [Noto Color Emoji 2.051](https://res.ebitengine.org/examples/NotoColorEmoji-Regular.ttf),
  name-table version `Version 2.051;GOOG;noto-emoji:20250818:e92753bfa55fd449e427d4d325f9c8c40408c74e`.
- [Twitter Color Emoji SVGinOT 15.1.0](https://github.com/13rac1/twemoji-color-font/releases/download/v15.1.0/TwitterColorEmoji-SVGinOT-15.1.0.zip):
  extract `TwitterColorEmoji-SVGinOT-15.1.0/TwitterColorEmoji-SVGinOT.ttf`.

The expected SHA-256 values of the TTF files are:

| Font | SHA-256 |
| --- | --- |
| Noto Color Emoji 2.051 | `7fb39738ab18f10612d6f4595e2e8e47a0afdf34738460442d99cd0c344a4d90` |
| Twitter Color Emoji SVGinOT 15.1.0 | `42d4b2a827c9ed557dff88c2f4723cb93985003a81dee76797324d66d04a57b8` |

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

## Native production reference checks

`TestSVGProductionReference` compares public glyph images and `text.Draw` with
librsvg (`rsvg-convert`), using the Noto gradient numeral and Twemoji even-odd
sunglasses records above in a synthetic font with 2048 units per em. It rejects
outline fallback.
The four cases cover 24- and 96-pixel em sizes; the small cases use a preceding
space to exercise fractional layout. Font origins follow text quantization;
reference placement does not use the renderer's computed ink bounds.

Production uses native-size vector AA unchanged. Only the independent reference
uses 8x rendering and CPU box averaging. Occupied premultiplied RGBA mean error
must be at most 8, with no channel error above 64. These are regression bounds,
not pixel equivalence: the tiny Noto numeral exhibits discrete native coverage
steps, including weak edge coverage absent from production. Inspection at native
and magnified sizes found edge differences; an unaveraged librsvg comparison
showed similar differences. These two single-paint records do not validate
coincident overpainting, adjoining-fill seams, or full-font compatibility.

Run one small case first, from the repository root:

```sh
go test -p 1 -count=1 -run '^TestSVGProductionReference/noto-3330-3330/24$' -v ./text/v2
```

On Linux prefix with `xvfb-run -a`. The test skips if `rsvg-convert` is absent.
Set `EBITENGINE_SVG_RENDER_REPORT_DIR` to an existing directory to retain native
production and averaged reference PNGs. Each comparison uses two 160x160 GPU
destinations and one small glyph image; the reference is at most 1280x1280.
Run sequentially and measure process memory before adding cases. Resource counts
do not bound physical GPU memory or engine allocation overhead.

### Original-font reference checks

`TestSVGRealFontReference` loads one of the pinned original fonts selected by
`EBITENGINE_SVG_ACCEPTANCE_FONT`, verifies its SHA-256, and renders Noto's flame
or Twemoji's window through normal `text.Draw`. Public glyph-image identity is
checked against the SVG cache so another color format or outline fallback cannot
count as SVG coverage. The checked-in, unmodified standalone SVG record supplies
the independent reference. Noto uses 1024 units per em; Twemoji uses 2048.

```sh
EBITENGINE_SVG_ACCEPTANCE_FONT=/path/font.ttf \
  go test -p 1 -count=1 -run '^TestSVGRealFontReference$' -v ./text/v2
```

Use the anchored `-run` shown above: this environment variable also enables
`TestSVGFontInventory`. Do not export it for an unfiltered package test run.

The two cases per font cover 24-pixel fractional layout and 96-pixel layout.
They use the same occupied-channel mean bound of 8, with a maximum channel error
bound of 96 for these layered records. These bounds permit native coverage and
composition-edge differences; they are not golden pixels or full SVG quality
acceptance. Better composition can reduce the errors without changing the tests.
On macOS 27.2, Metal and OpenGL produced the same comparison metrics against
librsvg 2.62.3/Cairo 1.18.4:

| Original font / glyph | Em size | Mean channel error | Maximum channel error |
| --- | ---: | ---: | ---: |
| Noto 2.051 / flame | 24 | 2.213 | 38 |
| Noto 2.051 / flame | 96 | 0.675 | 47 |
| Twemoji 15.1.0 / window | 24 | 4.592 | 66 |
| Twemoji 15.1.0 / window | 96 | 0.736 | 69 |

### Observed native limitations

Native-size and magnified inspection of the window exposes coincident-edge
overpainting: an edge has production alpha 191 versus reference alpha about 128.
Two half-covered opaque paints composed after coverage resolution yield
three-quarter alpha. An analytic pair of adjoining half-covered rectangles also
produces alpha 191 where their union should be opaque. Isolated groups do not
restore the lost coverage correlation. These observations confirm the documented
native-AA limitation; no wrong edge value is used as a golden expectation.

Native embedded images now use premultiplied area prefiltering
when their projected texel axes shrink, followed by clamped bilinear sampling.
The raster footprint is intersected with its image viewport before native path
coverage is resolved. Shader coordinates come from destination pixel centers,
including for ink-cropped output. Point-sampled reference plans explicitly retain
nearest sampling and their original viewport membership. Native magnification
intentionally uses bilinear smoothing; SVG `image-rendering` modes are not
implemented by this change.

The 64x64 one-texel checkerboard mapped to 8x8 now gives interior
premultiplied `(128,128,128,255)` through normal `text.Draw`. A transparent-red /
opaque-blue checkerboard gives `(0,0,128,128)`, without a red fringe. The restored
`(48.5,48.5)` translation in `TestGlyphInkAllocation` compares full and cropped
output exactly. Additional image checks cover a subpixel-width raster, half-pixel
edge coverage, one source reused at different reductions, meet/slice/none,
clipping, opacity and negative origins.

Only one filtered upload per decoded source is retained within a draw; replacing
its reduction releases the old upload. Upload dimensions never exceed the decoded
source. Filtering allocates only the reduced RGBA output and fixed-size
per-pixel integer accumulators, with no full-image floating-point intermediate.
The shared paint surface stays on the native output grid and is released
with other scratch images on return. Native image draws charge both the additional
paint pass and up to four source-texel visits per source pixel to the existing
cumulative work budget, including
repeated instances. Final glyph cache limits and native output dimensions are
unchanged. The source reduction averages axis-aligned texel areas, but the complete
transform is not an exact destination-pixel area integrator or anisotropic filter;
strong shear and fine detail under arbitrary affine transforms still need wider
quality evaluation.

### Sampling verification (2026-11-02)

- macOS ARM64, Go 1.27.0: renderer tests and SVG text integration checks passed on
  Metal and OpenGL in small serial batches. External original-font inventory and
  reference tests were not rerun. Those reports describe the baseline, not a new
  full inventory of this change.
- Linux ARM64: binaries from this checkout ran in Debian 13 under `xvfb-run`,
  with software OpenGL (Mesa 25.0.7), in the existing Podman VM (Linux 7.1.3).
  Renderer and SVG text batches passed. Independent librsvg and externally supplied
  font checks were skipped when unavailable. The committed baseline reproduced
  the checkerboard defects here, but did not reproduce the original nearest-filter
  crop failure on this driver.
- Windows 11 Pro ARM64 (10.0.26200), Parallels, DirectX: native image, crop,
  gradient-paint and selected normal-text checks passed in the logged-in console
  session. Parallels' non-interactive command session cannot run these graphics
  tests (`GetCursorPos` needs an interactive window station). A temporary interactive
  scheduled task was removed after testing, and the task-started VM was stopped.

Representative commands, run from the repository root:

```sh
go test -run '^Test(Native|GlyphInk)' ./text/v2/internal/opentypesvg/svgrender
go test -run '^TestSVGImageMinification$' ./text/v2
# Linux graphics commands must be prefixed with xvfb-run -a.
```

macOS processes used a 384 MiB RSS/physical-footprint watchdog, a 60-second
supervisor deadline, `GOMAXPROCS=2` and `GOMEMLIMIT=96MiB`. A combined OpenGL
renderer run exceeded that watchdog; smaller serial batches completed under it.
Linux disposable containers used a 384 MiB memory/swap cap and per-process
55-second deadlines. Windows used a 384 MiB working-set/private-byte watchdog
and 50-second process deadlines; sampled test peaks were below 100 MiB. These
measurements exclude some driver/system allocations and sampling can miss peaks.
No full-font sweep ran concurrently with a VM. The pre-existing Podman VM was left
running; temporary containers and Windows guest test files were removed.

The current-code analytic composition diagnostic still produces
`(64,0,128,191)` for coincident and adjacent half-covered red/blue rectangles.
Correct coincident blue is `(0,0,128,128)`; the adjoining union is
`(128,0,128,255)`. Image filtering does not restore vector coverage correlation.
General correction requires retaining coverage information across paint operations;
changing source-over blending or group isolation alone cannot distinguish these
two cases. This remains an acceptance limitation, alongside the broader
performance/resource evaluation. Legacy SVG removal remains deferred.

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
`Complete` means the requested inventory slice completed, not acceptance.
Decode failures are recorded and their glyph selections are still classified. A deferred report
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

### Bounded batch protocol

`EBITENGINE_SVG_ACCEPTANCE_BATCH_COUNT` selects 1..8 (glyph, size) cases.
`EBITENGINE_SVG_ACCEPTANCE_BATCH_START` is their zero-based starting index
(default 0) and requires a batch count. Indices follow document, size, range,
glyph order at sizes 24 and 96. Only the selected cases are prepared and only
their documents are decoded for inventory metadata. Omit both variables for
the existing full preparation inventory. The public text sample must run
separately from a batch.

Compile once from the repository root, keeping the binary unchanged during a
run:

```sh
GOMAXPROCS=2 go test -p 1 -c -o /tmp/svg-text.test ./text/v2
```

Run each batch in a fresh process, from `text/v2`, with an existing, distinct
report directory. For example, the child command for four GPU cases is:

```sh
GOMAXPROCS=2 \
EBITENGINE_SVG_ACCEPTANCE_FONT=/path/font.ttf \
EBITENGINE_SVG_ACCEPTANCE_REPORT_DIR=/path/batch-0 \
EBITENGINE_SVG_ACCEPTANCE_BATCH_START=0 \
EBITENGINE_SVG_ACCEPTANCE_BATCH_COUNT=4 \
EBITENGINE_SVG_ACCEPTANCE_GPU_GLYPHS=4 \
  /tmp/svg-text.test -test.run '^TestSVGFontInventory$' -test.timeout=60s
```

Use an external supervisor to monitor memory and stop the child on excessive
use; the Go harness has no memory watchdog or persistent checkpoint manager.
Run one child at a time, without concurrent validation or VMs. On Linux prefix
the child command with `xvfb-run -a`. A Go heap limit alone does not constrain
GPU/driver memory, and sampled process monitoring can miss short peaks.

The JSON report includes `Protocol`, `Start`, `Limit`, `Next`, `Total`, `Backend`,
and each case's `Index`. `Complete` means only that the requested slice finished.
To resume, retain a complete report with contiguous indices and an understood
exit status, then use its `Next` as the next starting index. Stop at `Next ==
Total`. Interrupted children, missing/truncated reports and incomplete slices
must be retried, not skipped. Preserve font and binary hashes, backend, limits
and environment alongside checkpoints so incompatible runs are not combined.
Classified preparation/rejection outcomes are completed observations even when
the test exits with failure; preserve them separately from successful cases.

GPU states distinguish `readback`, `valid-empty`, `preparation-failed`,
`dimension-limit`, `sample-limit`, and `failed`; preparation-only cases say
`not-requested`. Set the GPU count equal to the batch count to request every
case. Images over 256 pixels per axis remain explicitly unvalidated. Readback
is not a pixel oracle, and the inventory still bypasses format priority.
Completed batches do not establish full-font production or visual acceptance.
Keep fonts, binaries, reports, and external supervision tools outside the
repository.

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
and misses the image cache. `large` uses a 512-pixel em. Every drawing workload
rejects monochrome outline fallback. Readback
synchronizes each draw, so times measure latency, not asynchronous throughput.
Shader compilation on image misses is included. These short bounded probes are
not statistically robust performance acceptance measurements.

On 2026-11-02, eight-iteration Metal probes on an Apple M3 Pro with Go 1.27.0
measured synchronized latency of about 2.90 ms (`cold`), 0.80 ms (`warm`),
1.08 microseconds (`layout`), 1.59 ms (`colors`) and 1.93 ms (`large`). Each
workload ran alone under the 384 MiB watchdog. The allocation figures include
engine work during synchronized rendering and should not be interpreted as
allocations exclusively inside the glyph cache.

A temporary CPU-only area-filter probe reduced an opaque 2048x2048 source to
2047x2047, exercising the large near-unit reduction identified during review.
Two timed iterations measured about 78.5 ms per reduction and 16,769,088 allocated
bytes in two allocations per operation (the reduced RGBA pixel buffer and image
object). Source creation was outside the timer. The process stayed under the
384 MiB watchdog; this was an allocation/cost probe, not an allocator assertion in
the permanent tests or a GPU rasterization benchmark.

After the review changed the CPU prefilter, the native-image and crop tests were
rerun on Metal, OpenGL and Linux/Xvfb, and the normal `text.Draw` minification
check was rerun on Metal. The Windows results above precede that CPU-only filter
change; its shader and GPU coordinate path were unchanged. The diagnostic source
and supervisors stayed outside the repository. These small cost probes do not
complete many-glyph frame, document/plan churn or sustained memory-pressure
acceptance.
