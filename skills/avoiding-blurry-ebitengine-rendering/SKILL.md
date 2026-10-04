---
name: avoiding-blurry-ebitengine-rendering
description: >
  Use when creating an Ebitengine application or changing layout, text, UI,
  image scaling, or final presentation, and when diagnosing blurry rendering.
  Covers display-resolution rendering, font rasterization, sampling filters,
  coordinate transforms, and optional mixed-resolution layers. Apply even to
  pixel-art games with sharp UI, without requiring a low-resolution game screen.
---

# Avoiding blurry Ebitengine rendering

Render detail at the resolution where it will be displayed. Trace the complete
path from assets and glyph rasterization through intermediate images to the
display: any low-resolution stage can discard detail that later scaling cannot
recover. Hi-DPI support is one part of this, alongside sampling and transforms.

Preserve intentional pixel art and the requested presentation. Sharp rendering
does not mean disabling antialiasing or forcing nearest-neighbor filtering on
every image.

## Give the screen enough pixels

`LayoutF` receives outside dimensions in device-independent pixels (DIP) and
returns logical screen dimensions. The image passed to `Draw` uses those logical
dimensions, rounded up. Implementing `LayoutF` alone does not make it high
resolution: for display-resolution rendering, return outside dimensions times
`ebiten.Monitor().DeviceScaleFactor()`.

Adapt this pattern to the existing game; it uses `math` and `ebiten`:

```go
type Game struct {
	displayScale float64
	screenWidth  float64
	screenHeight float64
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("Layout is not called when LayoutF is implemented")
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	g.displayScale = ebiten.Monitor().DeviceScaleFactor()
	g.screenWidth = math.Max(1, outsideWidth*g.displayScale)
	g.screenHeight = math.Max(1, outsideHeight*g.displayScale)
	return g.screenWidth, g.screenHeight
}
```

`Layout` is still required by `ebiten.Game`. Keep the floating-point dimensions
for transforms; use `screen.Bounds()` when actual image bounds are needed.
The image passed to `Draw` is Ebitengine's intermediate game screen; the engine
then presents it to the final display. Matching display resolution avoids
enlarging a lower-resolution game screen in that final step.
Read the device scale during layout so a supported display-scale change can be
reflected. Do not cache it only at startup or enable the browser-specific
`RunGameOptions.DisableHiDPI` when native-resolution rendering is desired.

## Choose where each layer is rendered

- **Display-resolution scene and UI:** Draw directly into the high-resolution
  screen. Transform world or DIP coordinates into screen coordinates, and
  rasterize scalable content at that resolution. No extra game offscreen is
  required.
- **Fixed design coordinates with sharp rendering:** Keep the simulation and
  layout in design units, but transform geometry into the high-resolution
  screen. Fixed coordinates do not require a fixed-resolution raster image.
- **Intentional pixel art or an independently sized layer:** Render that layer
  into a reusable offscreen at its chosen resolution, composite it to the
  high-resolution screen, then draw sharp UI and text at screen resolution.
  This is an optional architecture, not a requirement for every game.

For an aspect-preserving fit from design or layer dimensions `(w, h)` into
logical screen dimensions `(W, H)`, use `s = min(W/w, H/h)` and offsets
`((W-w*s)/2, (H-h*s)/2)`. Scale, then translate. Choose integer scaling, cropping,
or stretching only when the intended presentation calls for it. Fill letterbox
areas as needed.

Reuse offscreens and recreate them only when their required resolution changes.
Clear them each frame unless retained content is intentional. Effects and cached
UI also need an explicit resolution policy: rendering a whole UI to a small
texture and enlarging it loses the benefit of the high-resolution screen.

## Rasterize text at its displayed size

With `text/v2.GoTextFace`, set `Size` in destination pixels and reuse the
`GoTextFaceSource`. For UI sized in DIP, multiply font size, positions, and line
spacing by the display scale. For text sized in design units, use the viewport's
design-to-screen scale instead. Do not multiply by device scale again when that
viewport scale already includes it.

For example, with a loaded `fontSource` and `text/v2` imported as `text`:

```go
face := &text.GoTextFace{
	Source: fontSource,
	Size:   20 * g.displayScale,
}
var op text.DrawOptions
op.GeoM.Translate(16*g.displayScale, 16*g.displayScale)
op.LineSpacing = 24 * g.displayScale
text.Draw(screen, "Score: 100", face, &op)
```

Do not also scale these glyphs by `g.displayScale` through `GeoM`. A geometry
transform enlarges rasterized glyphs; it does not request new glyph detail.
Likewise, enlarging a bitmap debug font cannot make it a high-resolution vector
font. Use bitmap fonts where their pixelated appearance is intentional.

Continuously changing `GoTextFace.Size` creates glyph images at many sizes.
For animated zoom, consider rendering at a suitable larger size and scaling down,
as the face's godoc suggests. Treat this as a quality/cache tradeoff, not a reason
to rasterize all static UI at a small size.

## Choose sampling and placement deliberately

- `FilterNearest` preserves hard texel boundaries. Integer magnification and
  aligned placement suit exact pixel art; fractional magnification can produce
  uneven pixel widths or shimmer.
- `FilterPixelated` preserves a pixelated appearance at non-integer scales.
  Use it for pixelated layers when that is the desired presentation. It cannot
  recover missing source detail or make downsampling lossless.
- `FilterLinear` suits smooth imagery and resampling, but enlarging a small
  source can look soft. Supply enough source resolution first. Linear
  minification can use mipmaps; do not disable them indiscriminately to pursue
  sharpness at the expense of aliasing.

Avoid repeated downscaling and upscaling through intermediate textures. Inspect
the final presentation transform as well as individual `DrawImage` options.
Use `FinalScreenDrawer` when custom final sampling is actually needed; the
legacy `SetScreenFilterEnabled` switch is deprecated. In this repository,
with the screen filter enabled (the default), `DefaultDrawFinalScreen` chooses
nearest for integer scales, linear for minification, and pixelated for other
magnification. Disabling the legacy screen filter forces nearest at every scale.

Check placement in destination pixels. Pixel art and thin axis-aligned features
may need deliberate alignment there; rounding design coordinates before scaling
does not guarantee alignment. Do not snap all motion or all text indiscriminately:
subpixel placement and antialiasing can be intentional. A filter change cannot
repair a low-resolution rasterization stage.

## Keep input consistent and verify

Ebitengine cursor and touch positions already use logical screen coordinates.
Do not multiply them by device scale again. Divide by display scale for UI
defined in DIP, or invert the viewport transform for design/game coordinates:
`x = (screenX-offsetX)/s`, `y = (screenY-offsetY)/s`. Account for additional camera
transforms and reject letterbox hits where appropriate. Calculate mapping from
current layout state in `Update`; do not depend on a previous `Draw`.
If a custom `FinalScreenDrawer` changes the supplied presentation transform,
account for that change in input mapping too; the engine's input conversion
uses its default presentation transform.

Check standard and high-density displays, fractional scaling, resizing, aspect
ratio changes, and monitor changes where supported. Inspect output at its actual
display size: enlarged previews can introduce their own blur. Verify intended
UI size, glyph detail, pixel-art presentation, thin edges, and hit targets.
If a display environment cannot be tested, state that limitation.

Compile application changes against the target Ebitengine version, especially
before adopting `FilterPixelated` or other newer APIs. Keep the resolution policy
explicit if performance requires lower-resolution effects or game layers.

## References

Use the target version's godoc and source for API and platform behavior.

- [Display scaling guide](https://github.com/tinne26/etxt/blob/main/docs/display-scaling.md):
  UI sizing and game-relative text scaling; its renderer examples use `etxt`.
- [Layout and presentation contracts](../../run.go): `LayoutFer`,
  `FinalScreenDrawer`, and `RunGameOptions`.
- [Final presentation](../../gameforui.go): `DefaultDrawFinalScreen`.
- [Filters](../../graphics.go) and [image options](../../image.go).
- [Text face sizing](../../text/v2/gotext.go) and [text drawing](../../text/v2/layout.go).
- [Input coordinates](../../input.go) and [high-DPI example](../../examples/highdpi/main.go).
