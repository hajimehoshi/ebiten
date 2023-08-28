# Ebitengine (v2)

[![Go Reference](https://pkg.go.dev/badge/github.com/hajimehoshi/ebiten/v2.svg)](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
[![Build Status](https://github.com/hajimehoshi/ebiten/actions/workflows/test.yml/badge.svg)](https://github.com/hajimehoshi/ebiten/actions?query=workflow%3Atest)

**A dead simple 2D game engine for Go**

Ebitengine (formerly known as Ebiten) is an open source game engine for the Go programming language. Ebitengine's simple API allows you to quickly and easily develop 2D games that can be deployed across multiple platforms.

* [Website (ebitengine.org)](https://ebitengine.org)
* [API Reference](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
* [Cheat Sheet](https://ebitengine.org/en/documents/cheatsheet.html)
* [Awesome Ebitengine](https://github.com/sedyh/awesome-ebitengine)

![Overview](https://ebitengine.org/images/overview2.png)

## Platforms

* [Windows](https://ebitengine.org/en/documents/install.html?os=windows) (No Cgo required!)
* [macOS](https://ebitengine.org/en/documents/install.html?os=darwin)
* [Linux](https://ebitengine.org/en/documents/install.html?os=linux)
* [FreeBSD](https://ebitengine.org/en/documents/install.html?os=freebsd)
* [Android](https://ebitengine.org/en/documents/mobile.html)
* [iOS](https://ebitengine.org/en/documents/mobile.html)
* [WebAssembly](https://ebitengine.org/en/documents/webassembly.html)
* Nintendo Switch
* Xbox

For installation on desktops, see [the installation instruction](https://ebitengine.org/en/documents/install.html).

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
  * [colorm](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/colorm)
  * [ebitenutil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/ebitenutil)
  * [inpututil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/inpututil)
  * [mobile](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/mobile)
  * [text](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/text)
  * [vector](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/vector)

## Community

- [Discord](https://discord.gg/3tVdM5H8cC)
- `#ebitengine` channel in [Gophers Slack](https://blog.gopheracademy.com/gophers-slack-community/)
- [GitHub Discussion](https://github.com/hajimehoshi/ebiten/discussions)
- [`r/ebitengine` in Reddit](https://www.reddit.com/r/ebitengine/)

## License

Ebitengine is licensed under Apache license version 2.0. See [LICENSE](LICENSE) file.

[The Ebitengine logo](https://ebitengine.org/images/logo.png) by Hajime Hoshi is licensed under [the Creative Commons Attribution-NoDerivatives 4.0](https://creativecommons.org/licenses/by-nd/4.0/).
