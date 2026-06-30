---
name: test-ebitengine-app
description: >
  Use this skill to test or debug a running Ebitengine app (anything
  implementing ebiten.Game) headlessly and programmatically — no visible
  window and no changes to the app's source. It launches the app as a
  vmhost guest, drives it through its ticks faster than real time, injects
  keyboard / mouse / touch / gamepad / text input, reads the rendered frame
  back as pixels to assert on or dump as a PNG, and observes the audio the
  app plays. Use when you need input injection, deterministic multi-tick
  runs, golden-image checks, or audio assertions — for example to
  reproduce an input-dependent bug, verify behavior after a sequence of
  clicks or key presses, check that the right sound plays, capture a single
  rendered frame, or let an AI agent drive an app end-to-end.
allowed-tools: Read, Edit, Write, Bash
---

# test-ebitengine-app skill

*Targets ebiten commit `c8db8fd6d` (2026-06-30), verified against it.
`exp/vmhost` is experimental, so if its API has moved since, the driver and
the snippets here may need updating.*

Drive an Ebitengine app from a small **host** program that runs it as a
**guest** of the `exp/vmhost` package. The host controls the guest's
ticks however it likes — one at a time, many per host frame, or
free-running — injects input, reads back rendered frames, and observes the
audio the app plays, all from a hidden window. The app under test is
unmodified.

## How it works (mental model)

Only the host touches a GPU or display; the guest is fully headless.

- **Guest** = the app under test. Its `ebiten.RunGame` dials the host's
  **endpoint** — the host listener's address as a URL, e.g.
  `unix:///path/to/socket` or `tcp://127.0.0.1:PORT` (the driver builds one
  from its listener with `vmhost.EndpointURLFromAddr`) — and connects over
  that socket *instead of opening a window*, forwarding its graphics
  commands rather than rendering. The endpoint reaches the guest one of two
  ways: built with `-tags ebitenginevm` it reads the
  `EBITENGINE_VM_ENDPOINT` environment variable (no source change — this is
  what the driver uses); or, with no build tag, the app's code sets
  `RunGameOptions.VMGuestEndpoint`.
- **Host** = the small driver you run. It is itself an `ebiten.Game`
  with `ebiten.SetWindowVisible(false)`, so it has no visible window. It
  replays the guest's draw commands on the real GPU, and exposes a
  `vmhost.GuestSession` to step the guest (`AdvanceTicks`/`AdvanceFrame`/
  `WaitFrame`/`CompositeFrame`), inject input, read pixels back, and observe
  the audio it plays.

A host driver lives at
[driver/main.go](.agents/skills/test-ebitengine-app/driver/main.go). Treat it
as a **starting-point template**, not a finished tool: copy it and adapt the
input script, assertions, and screen size to your case. It runs as-is only for
the trivial screenshot (recipe step 1); it is a skill asset, not a stable
command, so don't depend on its flags.

## When to use

- You need to **inject input** (keys, mouse, wheel, runes, touches,
  gamepads) and observe how the app responds — bugs that only reproduce
  after a click/drag/keystroke sequence.
- You need to **fast-forward to a state and assert it** — advance N ticks
  deterministically (compressed, so a long run is near-instant, not
  real-time), then check the resulting frame. Interleave to step frame by
  frame when you need every one.
- You want a **golden-image / pixel assertion** without a human running
  the app and pasting a screenshot.
- You just want a **screenshot** of the app — run the driver with no input
  script and dump the frame after N ticks (recipe step 1).
- You want to **check audio** — verify the app plays the expected sound, at
  the expected sample rate and volume, as raw PCM per player (un-mixed),
  without playing anything.

## When NOT to use

- You're testing pure logic with no rendering — write an ordinary Go
  test.
- Target platform is android / ios / js — `vmhost` is desktop-only.

## Recipe

Everything here is built on the driver at
[driver/main.go](.agents/skills/test-ebitengine-app/driver/main.go): step 1
runs it as-is, steps 2–3 copy and adapt it. The Go snippets below are excerpts
of its `Update`, so `d.guest` (the `*vmhost.GuestSession`), `d.screen`, and
`*ticks` (the `-ticks` flag value) are the driver's own identifiers.

Run these examples from the **ebiten repo root** — the `-pkg ./examples/...`
paths below are repo-relative. To test your own app instead, run the driver
from your app's module and point `-pkg` at your package.

### 1. No input — just capture a frame

Run the shipped driver directly against any guest package:

```bash
go run ./.agents/skills/test-ebitengine-app/driver \
  -pkg ./examples/rotate -ticks 60 -out /tmp/frame.png
```

Flags: `-pkg` (guest package), `-ticks` (how many ticks to run before
dumping + exiting), `-out` (PNG path), `-w`/`-h` (logical screen size,
default 320×240).

### 2. With input — script the ticks

Copy the driver to a scratch dir, edit the `INPUT SCRIPT` block in
`Update`, run it, then remove the copy (a `.`-prefixed dir stays out of
`go build ./...`):

```bash
mkdir -p .vmdriver && cp .agents/skills/test-ebitengine-app/driver/main.go .vmdriver/
# edit .vmdriver/main.go — the INPUT SCRIPT block in Update
go run ./.vmdriver -pkg ./examples/paint -ticks 120 -out /tmp/frame.png
rm -rf .vmdriver
```

The injectors on `*vmhost.GuestSession`. Injected state is observed at the
**next** `AdvanceTicks` and persists until changed, so timing comes from
*where* you inject in the run — split the run into `AdvanceTicks` segments and
inject between them:

```go
// keyboard
d.guest.PressKey(ebiten.KeyArrowRight)
d.guest.ReleaseKey(ebiten.KeyArrowRight)
d.guest.TypeRune('a')                 // a typed character (input chars)
// mouse — coordinates are outside-screen device-independent pixels
d.guest.MoveCursor(160, 120)
d.guest.PressMouseButton(ebiten.MouseButtonLeft)
d.guest.ReleaseMouseButton(ebiten.MouseButtonLeft)
d.guest.ScrollWheel(0, -1)
// touch — id, then device-independent pixels
d.guest.PressTouch(0, 100, 100)
d.guest.MoveTouch(0, 120, 100)
d.guest.ReleaseTouch(0)
// gamepads — full snapshot each call; see vmhost.GamepadState
d.guest.UpdateGamepads([]vmhost.GamepadState{ /* ... */ })
```

Example — settle 30 ticks, then a one-tick click, then run out the rest — as
the `INPUT SCRIPT` block:

```go
d.guest.AdvanceTicks(30) // let the app settle
d.guest.MoveCursor(160, 120)
d.guest.PressMouseButton(ebiten.MouseButtonLeft)
d.guest.AdvanceTicks(1)  // one tick with the button held
d.guest.ReleaseMouseButton(ebiten.MouseButtonLeft)
d.guest.AdvanceTicks(*ticks)
```

### 3. Inspect the output

Read the PNG with the Read tool; it renders inline. For a programmatic
assertion, read pixels in `Update` after `CompositeFrame` and compare —
`d.screen.ReadPixels(buf)` gives premultiplied-alpha RGBA, 4 bytes per
pixel. The center pixel of a `w*h` screen is at
`4*((h/2)*w + w/2)`. For a golden check, dump once, eyeball it, then
compare bytes against the saved PNG on later runs.

## How the driver drives the guest

The whole run happens in one host `Update`:

```go
d.guest.AdvanceTicks(*ticks) // run every tick back-to-back, no real-time pacing
d.guest.AdvanceFrame()       // request the final frame
d.guest.WaitFrame()          // block until all queued ticks have run and it is rendered
d.guest.CompositeFrame()     // composite it into the host-owned screen image
```

`AdvanceTicks(*ticks)` queues every tick at once; `WaitFrame` then blocks
until they have all run and the final frame is rendered, so one wait covers
the lot. That order keeps capture deterministic — the screen reflects exactly
the run's end. Only the one frame is rendered, because a tick forwards no
rendering; only a frame does.

**This compresses wall-clock time** — the point of the VM. The guest is not
paced to the host's ~60 TPS, so a 1000-tick run finishes near-instantly
instead of taking ~16 real seconds. To capture intermediate frames or pace
input by hand, interleave instead: `AdvanceTicks(n)`, `AdvanceFrame`,
`WaitFrame`, `CompositeFrame`, repeat. `PendingTicks` and `WaitTick` track the
backlog; ticks coalesce, so queuing many is cheap.

`guest.Err()` becomes non-nil when the guest terminates, crashes, or times
out; a panicking guest prints its stack to stderr and the driver stops.

## Observing audio

The guest plays no local audio; instead each of its audio players is
exposed to the host as a separate, **un-mixed** `*vmhost.GuestAudioStream`.
The audio methods are goroutine-safe and not frame-bound (unlike
`CompositeFrame`/`Close`), so inspect them anywhere. Control state (which
players exist, playing flag, volume) updates as you `AdvanceTicks`; samples
are pulled from the guest on demand. Drop this into `Update` after the
`AdvanceTicks` call:

```go
rate := d.guest.AudioSampleRate()          // per-channel samples/sec; 0 until audio plays
streams := d.guest.AppendAudioStreams(nil) // one entry per guest player
for i, s := range streams {
    slog.Info("guest audio player", "i", i,
        "playing", s.IsPlaying(), "volume", s.Volume(), "position", s.Position())
    // Raw PCM: 32-bit little-endian floats, two interleaved channels, at `rate`.
    // Read pulls on demand; volume is reported but NOT applied to these samples.
    buf := make([]byte, 4096)
    n, _ := s.Read(buf) // 0 bytes while paused; io.EOF at the source's end
    slog.Info("read PCM", "bytes", n, "rate", rate)
}
```

A stream stays valid until the guest closes its player; reaching `io.EOF`
does not remove it (a seek-and-replay yields more samples). To actually
*hear* it rather than just inspect it, feed the stream to a host
`audio.Player` as its source — see [examples/vm/main.go](examples/vm/main.go).

## Gotchas

- **Host and guest must be the same ebiten version.** The host/guest
  protocol is version-locked. The driver builds the guest in your current
  module, so they match automatically — but a guest built against a
  different ebiten version (such as a prebuilt binary) fails the handshake.
- **The host still needs a display/graphics context.** It is *windowless*,
  not *displayless*: on macOS it works from a normal shell; on bare
  headless Linux CI you still need Xvfb or an EGL setup to create the
  hidden window. The guest needs neither.
- **`SetOutsideScreen` before the first `AdvanceTicks`**, and the image is
  in **physical** pixels (logical size × `DeviceScaleFactor`). The driver
  handles this; keep it if you change the screen size.
- **Wall-clock isn't deterministic.** Tick *count* is exact, but code
  using `time.Now()` still sees real time between ticks.
- **`exp/vmhost` is experimental** — its API may change.
- **Unix socket path length is limited** (~104 bytes on macOS), so the
  driver keeps its socket in a short temp dir. Unix sockets work on every
  desktop target, Windows included; a `tcp` loopback listener
  (`127.0.0.1:0`) is an equivalent alternative if a filesystem socket is
  inconvenient.

## In-repo alternative (Go test)

To exercise an app as a guest from a `go test` instead of a standalone
driver, mirror the package's own end-to-end tests:
[exp/vmhost/guest_test.go](exp/vmhost/guest_test.go) (the `startGuest` /
`tickAndFrame` helpers) and
[exp/vmhost/readpixels_test.go](exp/vmhost/readpixels_test.go). These use
`internal/testing`'s `MainWithRunLoop`, which is importable only inside
the ebiten module.
