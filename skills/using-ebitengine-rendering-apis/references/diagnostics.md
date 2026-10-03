# Rendering diagnostics and capture boundaries

## Separate a scene error from a sampling error

For black or incorrect output, inspect in this order:

1. Draw the scene without post-processing.
2. Draw the intermediate image directly with `DrawImage`.
3. Run a shader that only returns `imageSrc0At(src0Pos)`.
4. Enable individual passes, checking dimensions, origins, alpha, and uniforms.

Test an identity setting and a small known input before tuning effect strength.
Specify units: a one-pixel sampling offset is not a normalized offset to be
multiplied by resolution again. Record the actual module revision, backend,
renderer, dimensions, and a minimal failing program before attributing a result
to the engine. A workaround does not establish the cause.

Neither `NewImage` plus `Fill` nor synchronous `ReadPixels` is inherently
invalid. Do not prohibit supported APIs based on an unreproduced application
failure. Image initialization paths can have different costs and internal
states without constituting a public rendering restriction.

## Read pixels through the supported execution path

Synchronous `ReadPixels` may flush rendering and incur a GPU-to-CPU transfer.
Avoid per-object readback, but use it when an actual pixel assertion is needed.
Run graphics-backed tests under Xvfb on Linux as required by the repository.
Use the [headless skill](../../run-ebitengine-app-headless/SKILL.md) for the
host/guest workflow; its synchronization protocol is not interchangeable with
a custom asynchronous readback API.

If an extension provides asynchronous readback, establish when command
submission happens before waiting. A callback must not block on completion that
requires that callback to return. Give each pixel buffer one owner, retain it
until completion, and drain issued work on errors before stopping the loop.
Do not assume such an extension exists in this checkout.

`RunGame` and `RunGameWithOptions` must not be called twice in one process.
Compare multiple states in a single loop, or use separate processes when testing
initialization. Resource use after the loop ends is not a second capture setup.

## Test what the comparison claims

For renderers intended to support random access, compare the same requested
state/time under A→B→C and C→A→B. Keep execution order and duplicates; associate
results by state/time, not array position. Distinguish a sample ID, an output
frame index, absolute time, and scene-local time.

Repeated same-order process runs do not establish order independence. Fixed
seeds alone do not remove history-dependent clears or lazy resource creation.
Initialize required resources in a stable order where practical; dynamic text
needs an explicit cache policy rather than a claim that all future glyphs were
warmed. Ordinary stateful games require input replay or snapshots instead of
assuming `Draw(time)` reconstructs simulation state.

Compare raw RGBA or decoded images before lossy encoding. Report changed pixels,
maximum channel difference, sum of absolute differences, and the affected bounds.
Start with exact comparison for an unchanged environment. If a justified tolerance
is needed, declare it and inspect the differences; do not widen it until a failure
passes or attribute unexplained differences to GPU rounding. Small text loss can
be significant even when the total changed area is small.

Inspect final-size images at maximum overlap, completed text, and clipping
boundaries. Thumbnails and numeric summaries help locate problems but cannot
establish readability or artistic quality. Record when visual inspection was
not possible.
