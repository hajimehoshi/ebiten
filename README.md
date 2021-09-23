# Ebiten (v2)

[![Go Reference](https://pkg.go.dev/badge/github.com/hajimehoshi/ebiten/v2.svg)](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
[![Build Status](https://github.com/hajimehoshi/ebiten/workflows/test/badge.svg)](https://github.com/hajimehoshi/ebiten/actions?query=workflow%3Atest)
[![Go Report Card](https://goreportcard.com/badge/github.com/hajimehoshi/ebiten)](https://goreportcard.com/report/github.com/hajimehoshi/ebiten)

**A dead simple 2D game library for Go**

Ebiten is an open source game library for the Go programming language. Ebiten's simple API allows you to quickly and easily develop 2D games that can be deployed across multiple platforms.

* [Website (ebiten.org)](https://ebiten.org)
* [API Reference](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
* [Cheat Sheet](https://ebiten.org/documents/cheatsheet.html)

![Overview](https://ebiten.org/images/overview2.2.png)

## Platforms

* [Windows](https://ebiten.org/documents/install.html?os=windows) (No Cgo!)
* [macOS](https://ebiten.org/documents/install.html?os=darwin)
* [Linux](https://ebiten.org/documents/install.html?os=linux)
* [FreeBSD](https://ebiten.org/documents/install.html?os=freebsd)
* [Android](https://ebiten.org/documents/mobile.html)
* [iOS](https://ebiten.org/documents/mobile.html)
* [WebAssembly](https://ebiten.org/documents/webassembly.html)
* Nintendo Switch™

Note: Gamepads and keyboards are not available on iOS.

For installation on desktops, see [the installation instruction](https://ebiten.org/documents/install.html).

## Features

* 2D Graphics (Geometry and color transformation by matrices, Various composition modes, Offscreen rendering, Text rendering, Automatic batches, Automatic texture atlas, Custom shaders)
* Input (Mouse, Keyboard, Gamepads, Touches)
* Audio (Ogg/Vorbis, MP3, WAV, PCM)

## Packages

* [ebiten](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
  * [audio](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio)
    * [mp3](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/mp3)
    * [vorbis](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/vorbis)
    * [wav](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/wav)
  * [ebitenutil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/ebitenutil)
  * [inpututil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/inpututil)
  * [mobile](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/mobile)
  * [text](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/text)

## Community

- [Discord](https://discord.gg/3tVdM5H8cC)
- `#ebiten` channel in [Gophers Slack](https://blog.gopheracademy.com/gophers-slack-community/)
- [GitHub Discussion](https://github.com/hajimehoshi/ebiten/discussions)
- [`r/ebiten` in Reddit](https://www.reddit.com/r/ebiten/)

## License

Ebiten is licensed under Apache license version 2.0. See [LICENSE](LICENSE) file.
