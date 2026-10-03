# Text coordinates, bounds, and caches

Use `github.com/hajimehoshi/ebiten/v2/text/v2` when shaping is needed. An ASCII
per-character renderer is not a general substitute for ligatures, combining
characters, bidirectional text, or font fallback.

## Construct faces explicitly

```go
// Imports: bytes and github.com/hajimehoshi/ebiten/v2/text/v2 as text.
source, err := text.NewGoTextFaceSource(bytes.NewReader(fontBytes))
if err != nil { return err }
face := &text.GoTextFace{Source: source, Size: 64}
small := *face
small.Size = 32
```

Reuse the source. Cache a bounded set of sizes and variation settings for the
actual vocabulary. Continuously changing size or baking every variation setting
can generate many glyph images. For animated scaling, consider rasterizing at
the largest intended display size and transforming the cached result. Measure
memory and quality rather than prescribing a universal raster size or axis step.
See [display-size rendering](../../avoiding-blurry-ebitengine-rendering/SKILL.md).

When using `golang.org/x/image/font/opentype.NewFace`, the owner closes the face
after its last CPU rasterization. If cache misses may still rasterize glyphs,
keep the face alive. Do not assume a face can be shared concurrently; check its
implementation or use separate ownership. GPU image lifetime is a separate issue.

## Layout bounds are not ink bounds

For default horizontal left-to-right layout and default alignment, `text.Draw`
anchors its layout region at `(0,0)` before `GeoM`. This is different from the
baseline origin used by `x/image/font.Drawer`.

`text.Measure` is not an ink-bounds query. For nonempty horizontal text, height
is `(lineCount-1)*lineSpacing + HAscent + HDescent`; empty text has zero size.
Set `DrawOptions.LineSpacing` consistently for multiple lines. Other alignments
and vertical writing require their own anchor interpretation.

With `x/image/font`, bounds, advances, and kerning use `fixed.Int26_6`:

```go
advancePx := float64(advance) / 64
leftPx := bounds.Min.X.Floor()
rightPx := bounds.Max.X.Ceil()
```

Convert before using values as pixels. An `image.Rectangle`, in contrast,
already has integer pixel coordinates. Include each glyph's pen position when
unioning ink bounds; otherwise overlapping unpositioned glyph bounds do not
represent the laid-out word. Preserve negative bearings and padding.

Use the positioned ink union to center an isolated label. Keep shared baselines
for rows of text rather than independently centering every row's ink. A panel
can use the positioned ink union plus padding, or reserve ascent/descent space
and enlarge it for ink overhang. Verify fallback fonts, descenders, longest
labels, outline widths, and shadows; a fixed padding ratio cannot guarantee fit.

## Transform the layout, not just each glyph bitmap

For a word scaled about a common anchor, both glyph positions and glyph shapes
must scale. With each glyph's bitmap origin already positioned in layout space:

```go
op := &ebiten.DrawImageOptions{}
op.GeoM.Translate(glyphX-anchorX, glyphY-anchorY)
op.GeoM.Scale(scale, scale)
op.GeoM.Translate(centerX, centerY)
dst.DrawImage(glyphImage, op)
```

Scaling each bitmap about its own center while retaining full-size advances
shrinks the letters but leaves wide gaps. Local glyph animation may be applied
before the shared transform; screen-pixel offsets belong after it. Check the
smallest and largest scales and the final held layout at native resolution.

## Sources

- [Face construction and size contract](../../../text/v2/gotext.go).
- [Source construction](../../../text/v2/gotextfacesource.go).
- [Layout anchors](../../../text/v2/layout.go) and
  [measurement](../../../text/v2/text.go).
