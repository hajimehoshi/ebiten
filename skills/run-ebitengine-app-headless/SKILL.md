---
name: run-ebitengine-app-headless
description: >
  Use this skill to run an Ebitengine app (anything implementing
  ebiten.Game) headlessly and programmatically — no visible window and
  no changes to the app's source — to test, debug, screenshot, or
  otherwise exercise it. It launches the app as a vmhost guest, drives it
  through its ticks faster than real time, injects keyboard / mouse /
  touch / gamepad / text input, reads the rendered frame back as pixels
  to assert on or dump as a PNG, and observes the audio the app plays.
  Use when you need input injection, deterministic multi-tick
  runs, golden-image checks, or audio assertions — for example to
  reproduce an input-dependent bug, verify behavior after a sequence of
  clicks or key presses, check that the right sound plays, capture a single
  rendered frame, or let an AI agent drive an app end-to-end.
allowed-tools: Read, Edit, Write, Bash
---

# run-ebitengine-app-headless skill

*Targets ebiten commit `a884b6732` (2026-07-14), verified against it.
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

- **Guest** = the app under test. Its `ebiten.RunGame` connects to the host
  over a socket *instead of opening a window*, forwarding its graphics
  commands rather than rendering.
- **Host** = the small driver you run. It is itself an `ebiten.Game`
  with `ebiten.SetWindowVisible(false)`, so it has no visible window. It
  replays the guest's draw commands on the real GPU, and exposes a
  `vmhost.GuestSession` to step the guest (`AdvanceTicks`/`AdvanceFrame`/
  `WaitFrame`/`CompositeFrame`), inject input, read pixels back, and observe
  the audio it plays.

## The endpoint

The guest reaches the host through an **endpoint** — the host listener's
address as a URL, e.g. `unix:///path/to/socket` or `tcp://127.0.0.1:PORT`
(the driver builds one from its listener with `vmhost.EndpointURLFromAddr`).
The endpoint reaches the guest one of two ways:

- Built with `-tags ebitenginevm`, the guest reads the
  `EBITENGINE_VM_ENDPOINT` environment variable (no source change — this is
  what the driver uses).
- With no build tag, the app's code sets `RunGameOptions.VMGuestEndpoint`.

## The driver

A host driver lives at
[driver/main.go](skills/run-ebitengine-app-headless/driver/main.go). Treat it
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
[driver/main.go](skills/run-ebitengine-app-headless/driver/main.go): step 1
runs it as-is, steps 2–3 copy and adapt it. The Go snippets below are excerpts
of its `Update`, so `d.guest` (the `*vmhost.GuestSession`), `d.screen`, and
`*ticks` (the `-ticks` flag value) are the driver's own identifiers.

Run these examples from the **ebiten repo root** — the `-pkg ./examples/...`
paths below are repo-relative. To test your own app in another module, copy
or invoke this driver from that module's root and point `-pkg` at the guest
package there. The driver imports `github.com/hajimehoshi/ebiten/v2/exp/vmhost`,
so the host and guest must resolve to the same Ebitengine version; use your
module's `go.mod` or `replace` directives to keep them aligned.

### 1. No input — just capture a frame

Run the shipped driver directly against any guest package:

```bash
go run ./skills/run-ebitengine-app-headless/driver \
  -pkg ./examples/rotate -ticks 60 -out /tmp/frame.png
```

Flags: `-pkg` (guest package), `-ticks` (how many ticks to run before
dumping + exiting), `-out` (PNG path), `-w`/`-h` (logical screen size,
default 320×240). In a restricted sandbox, `go run` may need a writable
`GOCACHE` or approval to use the normal Go build cache.

### 2. With input — script the ticks

Copy the driver to a scratch dir, edit the `INPUT SCRIPT` block in
`Update`, then run the copy (a `.`-prefixed dir stays out of
`go build ./...`; remove it when you are done):

```bash
mkdir -p .vmdriver && cp skills/run-ebitengine-app-headless/driver/main.go .vmdriver/
# edit .vmdriver/main.go — the INPUT SCRIPT block in Update
go run ./.vmdriver -pkg ./examples/paint -ticks 120 -out /tmp/frame.png
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

Tick counts are in the guest's own time units — see
[Ticks and TPS](#ticks-and-tps) to convert seconds of app time into
`AdvanceTicks` counts.

Example — use `-ticks` as the total run length, settle 30 ticks, hold a
click for one tick, then run the remaining ticks — as the `INPUT SCRIPT`
block. For shorter total runs, reduce the settle count first.

```go
d.guest.AdvanceTicks(30) // let the app settle
d.guest.MoveCursor(160, 120)
d.guest.PressMouseButton(ebiten.MouseButtonLeft)
d.guest.AdvanceTicks(1)  // one tick with the button held
d.guest.ReleaseMouseButton(ebiten.MouseButtonLeft)
if rest := *ticks - 31; rest > 0 {
	d.guest.AdvanceTicks(rest)
}
```

### 3. Many states — snapshot each one

The driver's `snapshot(i)` method renders the guest's *current* state and
writes it to a numbered PNG derived from `-out` (`frame.png` →
`frame_00.png`, `frame_01.png`, …). Call it between input segments in the
`INPUT SCRIPT` block to walk the app through N states and capture every
one — in practice the highest-value pattern (for example, clicking through
each page of a multi-page app and golden-comparing all of them):

```go
d.guest.AdvanceTicks(30) // let the app settle
for page := 0; page < pageCount; page++ {
	if page > 0 {
		d.guest.MoveCursor(nextButtonX, nextButtonY) // click "next page"
		d.guest.PressMouseButton(ebiten.MouseButtonLeft)
		d.guest.AdvanceTicks(1)
		d.guest.ReleaseMouseButton(ebiten.MouseButtonLeft)
		d.guest.AdvanceTicks(30)
	}
	if err := d.snapshot(page); err != nil {
		return err
	}
}
```

Each `snapshot` runs the `AdvanceFrame` / `WaitFrame` / `CompositeFrame`
sequence itself, so the numbered PNGs are exact frames; the final
`-out` frame at the end of `Update` is still written as usual.

### 4. Inspect the output

Inspect the PNG with the available image-viewing tool. For a programmatic
assertion, read pixels in `Update` after `CompositeFrame` and compare —
`d.screen.ReadPixels(buf)` gives premultiplied-alpha RGBA, 4 bytes per
pixel. Input coordinates and `-w`/`-h` are logical device-independent pixels,
but `d.screen` and the PNG are physical pixels (`logical × DeviceScaleFactor`;
for example, a 320×240 logical screen can produce a 640×480 PNG on a 2x
display). Use `b := d.screen.Bounds()` for assertion dimensions; the center
pixel of that physical `w*h` image is at `4*((h/2)*w + w/2)`. For a golden
check, dump once, eyeball it, then compare bytes against the saved PNG on
later runs — see [Verifying a run](#verifying-a-run) for the pitfalls.

## Verifying a run

Judge success by the **presence of the output PNG** (or the driver's final
`wrote frame` log line) — never by the exit status observed through a
pipe. Don't pipe the driver's output through `tail -1` or send it to
`/dev/null`: when the driver fails, its actual one-line error (the
`fmt.Fprintln` in `main`) is easily discarded, leaving only `go run`'s
generic `exit status 1` with nothing to diagnose. Save the full output to
a log file and read it on failure:

```bash
go run ./.vmdriver -pkg ./examples/paint -ticks 120 -out /tmp/frame.png \
  > /tmp/run.log 2>&1 || cat /tmp/run.log
```

When byte-comparing captures with `cmp -s`, check that both files exist
first: a missing file (from a capture that silently failed) also reads as
a difference, masquerading as a pixel regression.

### Golden baselines across a code change

To golden-compare before/after a code change, capture the baseline from
the committed state via `git show <commit>:<file>` extraction or a
temporary commit — **not** `git stash` round-trips: the stash list is
repo-global and shared across worktrees, so a stray `git stash pop` can
grab an unrelated stash entry belonging to the user.

## How the driver drives the guest

The whole run happens in one host `Update`:

```go
d.guest.AdvanceTicks(*ticks) // run every tick back-to-back, no real-time pacing
d.guest.AdvanceFrame()       // request the final frame
d.guest.WaitFrame()          // block until all queued ticks have run and it is rendered
d.guest.CompositeFrame()     // composite it into the host-owned screen image; check the bool in assertions
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
`WaitFrame`, `CompositeFrame`, repeat. `PendingTicks` and `WaitTicks` track the
backlog; ticks coalesce, so queuing many is cheap.

`guest.Err()` becomes non-nil when the guest terminates, crashes, or times
out; a panicking guest prints its stack to stderr and the driver stops. The
driver also puts a deadline on accepting the guest connection so a guest that
never dials the host fails instead of hanging indefinitely.

## Ticks and TPS

A tick is one guest `Update` call, and `AdvanceTicks(n)` runs exactly n of
them. The guest's TPS never throttles or scales this — how many ticks to run,
and at what real-time rate if any, is entirely the host's choice.

TPS matters as a unit conversion. The guest's game logic is written assuming
`Update` runs TPS times per second of real time, so its timers, animations,
and cooldowns count in those units. `d.guest.RequestedTPS()` reports what the
game requested via `ebiten.SetTPS` (the standard 60 until changed). To
simulate a duration of app time, multiply seconds by that value: one second
in a 30-TPS game is `AdvanceTicks(30)`, not 60.

`RequestedTPS` can also return `ebiten.SyncWithFPS` (-1), meaning the game
ties its ticks to rendered frames rather than a fixed rate. There is no
seconds-to-ticks conversion then; treat a tick as a frame and choose counts
by observing the app.

## Observing audio

The guest plays no local audio; instead each audio stream it starts is
handed to the host through the `OnAudioStream` handler registered when the
session is created, as a separate, **un-mixed** `*vmhost.GuestAudioStream`.
The driver already wires this up: `d.onAudioStream` collects the streams and
`d.appendAudioStreams(nil)` hands back the still-open ones (dropping any the
guest has closed). The audio methods are
goroutine-safe and not frame-bound (unlike `CompositeFrame`/`Close`), so
inspect them anywhere; sample data is pulled from the guest on demand. Drop
this into `Update` once the ticks have run (e.g. after `WaitFrame`, so the
streams the run started have been delivered):

```go
rate := d.guest.AudioSampleRate()             // per-channel samples/sec; 0 until audio plays
for i, s := range d.appendAudioStreams(nil) { // one entry per still-open guest player
    slog.Info("guest audio player", "i", i,
        "playing", s.IsPlaying(), "volume", s.Volume(), "position", s.Position())
    // Raw PCM: 32-bit little-endian floats, two interleaved channels, at `rate`.
    // Read pulls on demand; volume is reported but NOT applied to these samples.
    buf := make([]byte, 4096)
    n, _ := s.Read(buf) // 0 bytes while paused; io.EOF once the guest closes/ends the source
    slog.Info("read PCM", "bytes", n, "rate", rate)
}
```

`OnAudioStream` fires once per stream, during `AdvanceTicks` and `WaitTicks`
on the goroutine calling them, when the guest starts it. The driver's handler
only stashes the handle; read the samples from `Update`, as above. A stream
stays valid after `io.EOF` (a seek-and-replay yields more samples); once the
guest closes its player, `IsClosed()` returns true and `Read` reports
`io.EOF` for good, so `appendAudioStreams` drops it — returning only the
guest's still-open streams. To actually *hear* it rather than just inspect it,
feed the stream to a host `audio.Player` as its source — see
[examples/vm/main.go](examples/vm/main.go).

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
- **Call `GuestSession.Close` from the host frame.** The driver closes the
  guest inside `Update`, after dumping the frame. Keep that shape if you copy
  the template; `cmd.Wait` can run after `ebiten.RunGame` returns. Losing the
  host ends the closed guest's `RunGame` without an error, so it exits 0 on
  its own — the driver waits for it rather than killing it, and treats a
  non-zero exit as a genuine guest crash.
- **Launches are occasionally flaky.** Back-to-back runs sometimes fail on
  the first attempt while an identical rerun succeeds (root cause not
  identified; possibly socket/handshake timing). Retry once before
  debugging.
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
