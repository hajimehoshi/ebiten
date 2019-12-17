# Ebiten

[![Build Status](https://travis-ci.org/hajimehoshi/ebiten.svg?branch=master)](https://travis-ci.org/hajimehoshi/ebiten)
[![GoDoc](https://godoc.org/github.com/hajimehoshi/ebiten?status.svg)](http://godoc.org/github.com/hajimehoshi/ebiten)
[![Go Report Card](https://goreportcard.com/badge/github.com/hajimehoshi/ebiten)](https://goreportcard.com/report/github.com/hajimehoshi/ebiten)

**A dead simple 2D game library in Go**

Ebiten is an open-source game library, with which you can develop 2D games with simple API for multi platforms in the Go programming language.

* [Website (ebiten.org)](https://ebiten.org/)
* [Cheat Sheet](https://ebiten.org/documents/cheatsheet.html)

![Overview](https://ebiten.org/images/overview.png)

## Platforms

* Windows (No Cgo!)
* macOS
* Linux
* FreeBSD
* Android
* iOS
* Web browsers (Chrome, Firefox, Safari and Edge)
  * [GopherJS](https://github.com/hajimehoshi/ebiten/wiki/GopherJS)
  * [WebAssembly](https://github.com/hajimehoshi/ebiten/wiki/WebAssembly) (Experimental)

Note: Gamepad and keyboard are not available on Android/iOS.

For installation on desktops, see [the installation instruction](https://ebiten.org/install.html).

## Features

* 2D Graphics (Geometry/Color matrix transformation, Various composition modes, Offscreen rendering, Fullscreen, Text rendering, Automatic batches, Automatic texture atlas)
* Input (Mouse, Keyboard, Gamepads, Touches)
* Audio (Ogg/Vorbis, MP3, WAV, PCM)

## Packages

* [ebiten](https://godoc.org/github.com/hajimehoshi/ebiten)
  * [audio](https://godoc.org/github.com/hajimehoshi/ebiten/audio)
    * [mp3](https://godoc.org/github.com/hajimehoshi/ebiten/audio/mp3)
    * [vorbis](https://godoc.org/github.com/hajimehoshi/ebiten/audio/vorbis)
    * [wav](https://godoc.org/github.com/hajimehoshi/ebiten/audio/wav)
  * [ebitenutil](https://godoc.org/github.com/hajimehoshi/ebiten/ebitenutil)
  * [inpututil](https://godoc.org/github.com/hajimehoshi/ebiten/inpututil)
  * [mobile](https://godoc.org/github.com/hajimehoshi/ebiten/mobile)
  * [text](https://godoc.org/github.com/hajimehoshi/ebiten/text)

## Community

### Slack

`#ebiten` channel in [Gophers Slack](https://blog.gopheracademy.com/gophers-slack-community/)

## License

Ebiten is licensed under Apache license version 2.0. See LICENSE file.
