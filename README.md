# Ebiten

[![PkgGoDev](https://pkg.go.dev/badge/github.com/hajimehoshi/ebiten)](https://pkg.go.dev/github.com/hajimehoshi/ebiten)
[![Build Status](https://github.com/hajimehoshi/ebiten/workflows/test/badge.svg)](https://github.com/hajimehoshi/ebiten/actions?query=workflow%3Atest)
[![Build Status](https://travis-ci.org/hajimehoshi/ebiten.svg?branch=master)](https://travis-ci.org/hajimehoshi/ebiten)
[![Go Report Card](https://goreportcard.com/badge/github.com/hajimehoshi/ebiten)](https://goreportcard.com/report/github.com/hajimehoshi/ebiten)

**A dead simple 2D game library for Go**

Ebiten is an open source game library for the Go programming language. Ebiten's simple API allows you to quickly and easily develop 2D games that can be deployed across multiple platforms.

* [Website (ebiten.org)](https://ebiten.org)
* [API Reference](https://pkg.go.dev/github.com/hajimehoshi/ebiten)
* [Cheat Sheet](https://ebiten.org/documents/cheatsheet.html)

![Overview](https://ebiten.org/images/overview1.12.png)

## Platforms

* [Windows](https://ebiten.org/documents/install.html?os=windows) (No Cgo!)
* [macOS](https://ebiten.org/documents/install.html?os=darwin)
* [Linux](https://ebiten.org/documents/install.html?os=linux)
* [FreeBSD](https://ebiten.org/documents/install.html?os=freebsd)
* [Android](https://ebiten.org/documents/mobile.html)
* [iOS](https://ebiten.org/documents/mobile.html)
* Web browsers (Chrome, Firefox, Safari and Edge)
  * [WebAssembly](https://ebiten.org/documents/webassembly.html)
  * [GopherJS](https://ebiten.org/documents/gopherjs.html)

Note: Gamepads and keyboards are not available on iOS.

For installation on desktops, see [the installation instruction](https://ebiten.org/documents/install.html).

## Features

* 2D Graphics (Geometry/Color matrix transformation, Various composition modes, Offscreen rendering, Fullscreen, Text rendering, Automatic batches, Automatic texture atlas)
* Input (Mouse, Keyboard, Gamepads, Touches)
* Audio (Ogg/Vorbis, MP3, WAV, PCM)

## Packages

* [ebiten](https://pkg.go.dev/github.com/hajimehoshi/ebiten)
  * [audio](https://pkg.go.dev/github.com/hajimehoshi/ebiten/audio)
    * [mp3](https://pkg.go.dev/github.com/hajimehoshi/ebiten/audio/mp3)
    * [vorbis](https://pkg.go.dev/github.com/hajimehoshi/ebiten/audio/vorbis)
    * [wav](https://pkg.go.dev/github.com/hajimehoshi/ebiten/audio/wav)
  * [ebitenutil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/ebitenutil)
  * [inpututil](https://pkg.go.dev/github.com/hajimehoshi/ebiten/inpututil)
  * [mobile](https://pkg.go.dev/github.com/hajimehoshi/ebiten/mobile)
  * [text](https://pkg.go.dev/github.com/hajimehoshi/ebiten/text)

## Community

### Slack

`#ebiten` channel in [Gophers Slack](https://blog.gopheracademy.com/gophers-slack-community/)

## License

Ebiten is licensed under Apache license version 2.0. See LICENSE file.
