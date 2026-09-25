---
name: efficient-ebitengine-rendering
description: >
  Use when writing or changing Ebitengine drawing code, arranging sprites or
  offscreen passes, or investigating rendering performance. Explains automatic
  draw batching, texture atlases, render dependencies, and how to inspect actual
  GPU commands. Apply during implementation, before performance problems appear.
---

# Efficient Ebitengine rendering

Design drawing around compatible successive commands and necessary render
passes. The number of Go drawing calls is not the number of GPU draw commands.
Preserve the intended image while improving batching.

## How texture atlases enable batching

A texture atlas is a large GPU texture containing multiple smaller images in
separate rectangular regions. Drawing an image samples its region of that
texture. Several sprites can therefore use the same GPU texture even though
their pixels and positions differ.

Ebitengine automatically packs eligible images into internal texture atlases
and adjusts their texture coordinates when drawing. A Go `*ebiten.Image` does
not necessarily own a separate GPU texture. Applications draw using each
image's coordinates; Ebitengine handles its placement within the atlas.

For example, successive draws of a player image and a tree image can merge into
one GPU command when both occupy the same atlas and the other drawing state is
compatible. The command samples different regions of the same texture for each
sprite. If their backing textures differ, the texture change splits the batch.
Sharing an atlas enables batching but does not remove the state and ordering
constraints described below.

A manually prepared sprite sheet is useful for asset organization but is not a
prerequisite for batching: separately loaded images can share an automatic
atlas. Atlas capacity, large images, and render-target usage can result in
separate backing textures. Placement can change as images are used; do not
assume every image shares one atlas or hard-code a portable atlas size.

### Unmanaged images

Most applications do not need unmanaged images. Leave `Unmanaged` at its default
`false`; opting out of automatic atlas management is a rarely needed tuning
option, including for offscreen render targets.

Set `Unmanaged: true` in `NewImageOptions` passed to `NewImageWithOptions`, or
in `NewImageFromImageOptions` passed to `NewImageFromImageWithOptions`, to keep
an image off the automatic atlas permanently.

Unmanaged images provide explicit control over atlas participation for
performance and memory tuning. Separate unmanaged images cannot share an
automatic atlas, so switching between them as sources splits batches. Repeated
draws from the same unmanaged image, including its subimages, can still batch
when the other state is compatible. Unmanaged does not mean batching is disabled.

Use unmanaged images only when a specific, measured performance or memory issue
justifies controlling atlas participation. A frequently updated render target
alone is not a reason to enable it. Verify the effect on commands and memory;
it is not a general performance switch.

## Keep compatible draws together

Successive `DrawImage` calls batch most readily when their destination, blend,
and filter match. Different source `*ebiten.Image` values can still batch when
they share an internal texture atlas. Different positions, rotations, scales,
and `ColorScale` values do not inherently prevent batching: these can be
represented in vertex data.

With linear minification, different transforms can select different mipmap
levels and therefore different backing source textures. This can split batches
even when the original source image and public drawing options match.

For `DrawTriangles`, also consider addressing and the other drawing options.
Do not assume that matching the public options guarantees one GPU command.
The internal command merger compares the destination, backing source textures,
shader, blend, and effective uniforms; buffer capacity can also split commands.
Changing a custom shader's uniforms between objects can therefore split draws
even when they use the same shader. Changes to clipping and source regions can
also affect effective state or backend work. Inspect the target version when
an exact batching condition matters.

- Group compatible draws only within ordering constraints. Overlapping sprites
  and alpha compositing generally depend on painter's order; even opaque
  overlapping sprites cannot be freely sorted by texture.
- Finish a render pass before switching destinations when dependencies allow.
- Reuse image and shader resources. Do not allocate a new image per sprite or
  frame to reduce the apparent number of drawing calls.
- Use `ColorScale` for ordinary tint and opacity. Do not introduce a color-matrix
  shader or varying uniforms for an operation vertex colors already express.
- Do not replace ordinary sprite calls with manual triangle assembly solely
  because there are many calls. First determine whether they already batch.

### Kage builtins can introduce implicit uniforms

Matching user-supplied uniforms is not sufficient for shader draws to batch.
Some Kage builtins read internal uniforms that describe the images used by each
draw. If those values differ between successive draws, they can split a batch
even when the shader and backing textures are the same.

- `imageSrcNOrigin()` and `imageSrcNSize()` read source-region uniforms. Different
  images or subimages on the same atlas can have different origins or sizes.
  In pixel-unit `DrawRectShader` calls without source image 0, `imageSrc0Size()`
  returns the requested rectangle size, so changing that size can also split
  batches when this builtin is used.
- `imageDstOrigin()` and `imageDstSize()` read destination-region uniforms.
- Texture-size builtins read backing-texture size uniforms. These values need
  not change when switching between regions on the same atlas.
- Safe sampling such as `imageSrc0At()` implicitly reads source-region origin
  and size to return transparent pixels outside the image. This introduces
  region-dependent uniforms even without an explicit origin or size query.
- Sampling other source slots can also read origins to convert coordinates
  from source 0's space, including with unsafe sampling.

The command queue prepends these internal uniforms, then
`FilterUniformVariables` zeroes unused uniform variables before the merger
compares the uniform data. Only retained dependencies matter; calling a builtin
does not invariably break batching. The current filtering tracks whole uniform
variables, so referencing one element of a source-region array retains the
entire array, not just that element.

In pixel-unit shaders, `imageSrc0UnsafeAt()` avoids the region bounds check and
its region uniforms. Legacy texel-unit sampling also depends on texture-size
uniforms to convert coordinates. Use unsafe sampling only when every sampled
position is guaranteed to stay inside the source image; otherwise it can sample
unrelated atlas content.
Do not remove bounds checks or image-coordinate calculations needed for
correctness merely to improve batching. Check actual commands using the debug
tag described below when evaluating a shader change.

## Account for offscreen passes

Images drawn into as render targets may be moved away from atlases used for
source images. Ebitengine can move eligible images back as their usage changes,
so a managed offscreen is not permanently excluded from source atlases.
Frequently redrawn offscreens should not be assumed to share the sprites'
source atlas.

An offscreen pass adds rendering and compositing work. Use one when it serves a
concrete purpose, such as a layer's resolution, an effect, or reusable content.
Creating an offscreen for each object can fragment source textures and defeat
batching. Reuse offscreens, resize them only when their required size changes,
and clear them when their previous contents should not persist.

Arrange dependencies so a layer is drawn before it is sampled. Avoid repeatedly
alternating between writing a source and drawing it into another image when the
same result can be produced by finishing the source first. Feedback effects may
need separate ping-pong images. Drawing an image onto itself, including through
subimages of the same backing image, is invalid.

Keep GPU readback (`At`, `ReadPixels`) out of per-object drawing loops. Readback
can force queued rendering to complete and transfer pixels to the CPU. Keep CPU
copies for CPU-side queries when appropriate. Consolidate necessary pixel
uploads; do not rebuild static image contents every frame.

## Verify the actual commands

Run the application with `-tags=ebitenginedebug` to inspect internal graphics
commands, for example `go run -tags=ebitenginedebug ./path/to/app`. Follow the
project's display or headless execution requirements when running it.

Inspect a representative frame and identify which destinations, source textures,
shaders, or state changes split compatible runs. Compare before and after under
the same scene. Include offscreen and final presentation passes when counting
work; a large sprite count can legitimately produce only a few commands.
The log describes internal commands, not necessarily individual backend draw
calls: one command can contain several destination regions that a backend draws
separately. Check the logged destination-region count when comparing work.

Verify the rendered result, especially overlap, transparency, clipping, and
effects. Fewer commands alone do not prove a speedup: overdraw, shader cost,
pixel count, uploads, and CPU work can dominate. Measure the relevant bottleneck
before adding more complicated rendering machinery.

## References

Use the target project's godoc and source for version-specific behavior. The
performance article contains older API names and examples.

- [Performance tips](https://ebitengine.org/en/documents/performancetips.html).
- [Image APIs](../../image.go): `DrawImage`, drawing options, uploads, readback.
- [Command merging](../../internal/graphicscommand/command.go):
  `CanMergeWithDrawTrianglesCommand`.
- [Command queue](../../internal/graphicscommand/commandqueue.go): effective
  uniforms, buffer limits, and enqueueing.
- [Kage builtin bridge](../../internal/graphics/shader.go): implicit image
  uniforms and sampling functions.
- [Shader uniform filtering](../../internal/shaderir/program.go):
  `FilterUniformVariables`.
- [Atlas implementation](../../internal/atlas): backing textures and allocation.
- [Mipmap selection](../../internal/mipmap/mipmap.go): source textures used for
  minification.
