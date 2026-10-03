---
name: using-ebitengine-rendering-apis
description: >
  Use before implementing or debugging Ebitengine image composition, vector
  drawing, text layout, or pixel capture. Covers alpha representations, clipping,
  current API selection, glyph bounds, resource ownership, and reproducible
  rendering diagnostics. For engine patches, also follow repository guidelines.
---

# Use the rendering contracts before adding effects

Check the application's resolved Ebitengine module and its public godoc before
copying code from a different version or fork. A remembered name that does not
compile is not evidence that an API was removed. Prefer documented replacements
for deprecated APIs.

Read the relevant contract at the point it is needed:

| Work | Read before implementation |
|---|---|
| Sprites, opaque panels, clipping, vector paths | Image and vector contracts below |
| Font caches, glyph transforms, bounds | [Text layout and resource lifetime](references/text-layout.md) |
| Shader inputs and post-processing | [Writing Kage shaders](../writing-kage-shaders/SKILL.md) |
| Large meshes and repeated draws | [Efficient rendering](../efficient-ebitengine-rendering/SKILL.md) |
| Scaling and display resolution | [Avoiding blurry rendering](../avoiding-blurry-ebitengine-rendering/SKILL.md) |
| Black output, capture differences, repeated evaluation | [Rendering diagnostics](references/diagnostics.md) |

Record the chosen coordinate space, alpha representation, resource owner, and a
small verification case with the implementation. A list of documents read is
not a substitute for applying their contracts.

## Image colors and vertex colors are different contracts

- Go's `image.RGBA` stores premultiplied values. White at alpha `a` is
  `color.RGBA{a, a, a, a}`. Storing `{255,255,255,a}` there for `a < 255`
  violates that representation.
- `image.NRGBA` stores straight values: white is
  `color.NRGBA{255,255,255,a}`. `NewImageFromImage` accepts this image and
  converts it; do not premultiply an NRGBA image yourself.
- `DrawTrianglesOptions.ColorScaleMode` defaults to
  `ColorScaleModeStraightAlpha`. If vertex colors are already premultiplied,
  select `ColorScaleModePremultipliedAlpha`. Do not infer the vertex convention
  from the image's storage convention.
- The default blend is source-over. `BlendSourceOver` covers the background
  with an opaque source; `BlendLighter` adds light and cannot use black to hide
  an object behind a panel.

For an unexpectedly transparent panel, check the source texture's alpha,
vertex alpha, other color multipliers, blend, geometric coverage, and drawing
order. At alpha 1, straight and premultiplied RGB are identical: missing RGB
premultiplication alone cannot explain an opaque panel becoming transparent.
Antialiased edge coverage is different from transparency in the panel interior.

```go
// Imports: image/color and github.com/hajimehoshi/ebiten/v2/vector.
// x, y, width, height are float32 values in destination coordinates.
vector.FillRect(dst, x, y, width, height,
    color.RGBA{0x0a, 0x0c, 0x10, 0xff}, true)
```

For arbitrary paths, use `vector.FillPath`; for strokes, `vector.StrokePath`.
`Path.AppendVerticesAndIndicesForFilling` is deprecated as of v2.9 in favor of
`FillPath`. The legacy stroke append method directs callers to `StrokePath`
or `Path.AddStroke`. Use `vector.Clockwise` or `vector.CounterClockwise` for
`Path.Arc`, rather than guessing a boolean or another package's enum.
A rectangle helper is convenient, not proof that triangle meshes are slower.

## Distinguish source crops from destination clips

A subimage retains its parent's coordinate bounds. However, drawing it with
`DrawImage` does not automatically place it at its original parent position.

```go
// Source crop: restore its original location only when that is intended.
r := image.Rect(100, 40, 200, 80)
crop := source.SubImage(r).(*ebiten.Image)
op := &ebiten.DrawImageOptions{}
op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
dst.DrawImage(crop, op)

// Destination clip: draw using the parent's destination coordinates.
clip := dst.SubImage(r).(*ebiten.Image)
clip.DrawImage(sprite, spriteOptions)
```

`Vertex.SrcX/SrcY` use source-image pixel coordinates, including a nonzero
subimage origin; they are not normalized UVs. Do not apply the source-crop
`DrawImage` placement rule to triangle sampling coordinates.

A subimage shares storage with its parent. Keep that storage alive while it is
used, and do not sample a destination from itself through an alias. Reuse
render targets and clear them when previous contents should not persist.

## API and environment boundaries

This checkout is the authority for API availability. Do not import fork-only
`Headless` or `ReadPixelsAsync` calls into applications targeting a version
without them. Use [the headless execution skill](../run-ebitengine-app-headless/SKILL.md)
for this repository's supported capture workflow.

Keep project-specific module replacements, compiler workarounds, FFmpeg
settings, and creative timing decisions in their owning project or workflow.
They are not Ebitengine rendering requirements.
