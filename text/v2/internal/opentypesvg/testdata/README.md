# SVG test data

Ten unmodified font SVG records and twelve focused fixtures for the
[SVG renderer](../README.md). Attribution and license texts are in
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).

## Sources and regeneration

- [Noto Color Emoji 2.051](https://res.ebitengine.org/examples/NotoColorEmoji-Regular.ttf),
  name-table version `Version 2.051;GOOG;noto-emoji:20250818:e92753bfa55fd449e427d4d325f9c8c40408c74e`.
- [Twitter Color Emoji SVGinOT 15.1.0](https://github.com/13rac1/twemoji-color-font/releases/download/v15.1.0/TwitterColorEmoji-SVGinOT-15.1.0.zip):
  use `TwitterColorEmoji-SVGinOT-15.1.0/TwitterColorEmoji-SVGinOT.ttf` from the archive.

Verify the TTF SHA-256 before regeneration or acceptance runs:

| Font | SHA-256 | Units per em |
| --- | --- | ---: |
| Noto Color Emoji 2.051 | `7fb39738ab18f10612d6f4595e2e8e47a0afdf34738460442d99cd0c344a4d90` | 1024 |
| Twitter Color Emoji SVGinOT 15.1.0 | `42d4b2a827c9ed557dff88c2f4723cb93985003a81dee76797324d66d04a57b8` | 2048 |

From this directory:

```sh
go run extract.go /path/NotoColorEmoji-Regular.ttf /path/TwitterColorEmoji-SVGinOT.ttf
```

The standard-library tool overwrites `real/*.svg` and generates an untracked
`inventory.json` with UPEM, glyph ranges and construct counts per unique SVG
document. Compare regenerated records with the checked-in files. Retain pinned
fonts offline because download contents can change; keep fonts and generated
reports outside the repository. License files are maintained separately.

`focused/shared-context.svg.gz` must decompress to the exact bytes of
`focused/shared-context.svg`. The JPEG fixture is original Apache-2.0 test data:
fill an 8×8 `image.RGBA` with `color.RGBA{R: 80, G: 160, B: 240, A: 255}`, encode
with the standard-library JPEG encoder at quality 100, and base64-encode it.

## Fixture coverage

Focused fixtures use viewport/UPEM 48 and glyph 1. Pixel expectations in the tests
use premultiplied RGBA with one-byte rounding tolerance. CPU style/geometry tests
check interpretation; GPU tests in `../svgrender` check rendering. See the
[package validation guide](../README.md#validation) for ordinary test commands.

### Real font records

| Record | Coverage |
| --- | --- |
| Noto 2–3 | Glyph selection and differently colored keycaps sharing paths |
| Noto 3330 | Quadratic numeral path and user-space linear gradient |
| Noto 3629 | Layered burst, radial gradient and negative-y coordinates |
| Noto 3682 | Transformed radial gradients and translucent flame highlight |
| Noto 3737 | Layered octagon, opacity and ink beyond the em square |
| Twemoji 692 | Nested transforms and layered basic shapes |
| Twemoji 839 | Even-odd sunglasses path and explicit clip-rule |
| Twemoji 1039 | Referenced group clip and painter order |
| Twemoji 1338 | Twelve user-space clips following referencing-path transforms |
| Twemoji 1355 | Window panes, translucent reflection and opaque crossbars |

### Focused fixtures

| Fixture | Coverage |
| --- | --- |
| css-classes | Stylesheet dependencies, glyph selection and opacity |
| group-identity | Identical shape/clip edges under group opacity |
| nested-groups | Painter order, nested half-opacity groups and intersecting clips |
| evenodd-clip | Matching-winding inner contour remains a hole |
| gradient-reference | Inherited stops, radial paint and definition-order independence |
| shared-context | Use-instance selection and inheritance, missing glyph, gzip equivalence |
| stroke-transform | Transformed stroke width, round caps and retained curves |
| palette | currentColor, palette fallback and palette alpha applied once |
| embedded-jpeg | Opaque encoded RGB image mapped into an SVG viewport |
| embedded-png | Red/blue texel mapping and image viewport bounds |
| restricted | Prohibited subtrees neither render nor execute |
| viewport-overflow | Negative ink survives; the root viewport is not a glyph clip |

## Native production reference checks

These tests compare public glyph images and normal `text.Draw` with independent
librsvg output (`rsvg-convert`). Production stays at native-size vector AA; only
the reference uses 8x rendering and CPU box averaging. Cases use 24- and 96-pixel
em sizes, including fractional layout, and reject fallback to another format.

- `TestSVGProductionReference` embeds the Noto numeral and Twemoji sunglasses
  records in a synthetic 2048-UPEM font. Occupied premultiplied-channel error
  limits are mean 8 and maximum 64.
- `TestSVGRealFontReference` verifies a pinned original font's hash and renders
  Noto's flame or Twemoji's window. Its limits are mean 8 and maximum 96.

From the repository root, start with one small case:

```sh
go test -p 1 -count=1 -run '^TestSVGProductionReference/noto-3330-3330/24$' -v ./text/v2

EBITENGINE_SVG_ACCEPTANCE_FONT=/path/font.ttf \
  go test -p 1 -count=1 -run '^TestSVGRealFontReference$' -v ./text/v2
```

Tests skip without `rsvg-convert`. Set `EBITENGINE_SVG_RENDER_REPORT_DIR` to an
existing directory to retain production/reference PNGs. Use the anchored test
names: exporting the font variable for an unfiltered run also enables inventory.

These are regression tolerances, not visual acceptance. Native AA still loses
coverage correlation between separate paints, causing coincident-edge fringes
and adjoining-fill seams; isolated groups do not fix this. Embedded-image tests
cover minification, premultiplied filtering and exact half-pixel crop consistency,
but strong shear and general affine sampling remain quality work. Near-unit
reduction of large embedded images can be expensive on cache misses. Full-font
visual and sustained performance acceptance remain incomplete. Rendering policy
and resource limits belong in the [package README](../README.md#rendering).

## Full-font inventory and bounded GPU checks

`TestSVGFontInventory` defaults to preparing SVG candidates at 24 and 96 pixels,
independently of production format priority. It writes `full-font.json` with
font/document hashes, preparation outcomes and optional realization outcomes.
It still uses the hidden graphics test lifecycle, even without GPU realization.

All graphics commands here, including inventory, require `xvfb-run -a` on Linux.
Run one process at a time with no concurrent sweeps or VMs. Use an external
memory/time watchdog: the harness has no memory watchdog, and a Go heap limit
does not constrain GPU/driver memory. Work limits are not physical-memory caps.
Keep fonts, binaries, reports and supervisors outside the repository.

From the repository root, with an existing report directory:

```sh
EBITENGINE_SVG_ACCEPTANCE_FONT=/path/font.ttf \
EBITENGINE_SVG_ACCEPTANCE_REPORT_DIR=/path/report \
  go test -p 1 -count=1 -run '^TestSVGFontInventory$' -v ./text/v2
```

| Optional variable (`EBITENGINE_SVG_ACCEPTANCE_` prefix) | Effect |
| --- | --- |
| `GPU_GLYPHS` | Request up to 1–8 realizations/readbacks; default 0 |
| `BATCH_COUNT` | Prepare only 1–8 (glyph, size) cases |
| `BATCH_START` | Zero-based start index; defaults to 0 and requires `BATCH_COUNT` |
| `TEXT` | Separate public-text check of at most eight glyphs; requires `GPU_GLYPHS` and cannot accompany a batch |

GPU images over 256 pixels per axis are skipped explicitly. `TEXT` checks SVG
cache identity and compares `text.Draw` with explicit glyph composition; choose
nonempty SVG glyphs without preferred bitmap/COLRv0 data. Neither readback nor
that self-comparison is an independent pixel oracle or exhaustive ligature test.

### Bounded batch protocol

Case indices follow document, size, range, glyph order. A batch prepares only its
selected cases and decodes metadata only for their documents. Omit both batch
variables for a full preparation inventory. Compile once from the repository
root and keep that binary fixed throughout a run:

```sh
GOMAXPROCS=2 go test -p 1 -c -o /tmp/svg-text.test ./text/v2
```

Run each batch in a fresh process **from `text/v2`**, with a distinct existing
report directory. This requests all four selected cases for GPU readback:

```sh
GOMAXPROCS=2 \
EBITENGINE_SVG_ACCEPTANCE_FONT=/path/font.ttf \
EBITENGINE_SVG_ACCEPTANCE_REPORT_DIR=/path/batch-0 \
EBITENGINE_SVG_ACCEPTANCE_BATCH_START=0 \
EBITENGINE_SVG_ACCEPTANCE_BATCH_COUNT=4 \
EBITENGINE_SVG_ACCEPTANCE_GPU_GLYPHS=4 \
  /tmp/svg-text.test -test.run '^TestSVGFontInventory$' -test.timeout=60s
```

The report includes `Protocol`, `Start`, `Limit`, `Next`, `Total`, `Backend` and
per-case `Index`. Resume only from a complete report with contiguous indices and
an understood exit status, using `Next`; finish when `Next == Total`. Retry
interrupted, missing, truncated or incomplete slices rather than skipping them.
Retain font/binary hashes, backend, limits and environment with checkpoints;
do not combine incompatible runs. Deferred reporting cannot survive termination.

Preparation distinguishes malformed, unsupported, resource-limited, valid-empty
and successful candidates. GPU results distinguish readback, valid-empty,
preparation-failed, dimension-limit, sample-limit, failed and not-requested.
Classified rejections are observations to retain even when the test exits with
failure. `Complete` means the requested slice finished, not that it passed;
`Realized` means readback completed, not that pixels or format selection are correct.

## Benchmarks

Run one workload per process, with at most 16 iterations:

```sh
EBITENGINE_SVG_ACCEPTANCE_BENCHMARK=1 go test -p 1 -run '^$' \
  -bench '^BenchmarkSVGProduction/warm$' -benchtime=16x ./text/v2
```

`cold` includes synthetic font construction/loading; `warm` uses a prewarmed
image cache; `layout` measures cached lazy layout; `colors` changes currentColor;
`large` uses a 512-pixel em. Drawing workloads reject outline fallback and use
readback to synchronize. Results measure short-run latency, not asynchronous
throughput, pure SVG rasterization cost or sustained memory behavior.
