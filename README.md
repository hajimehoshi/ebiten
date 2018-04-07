# Ebiten (海老天)

[![Build Status](https://travis-ci.org/hajimehoshi/ebiten.svg?branch=master)](https://travis-ci.org/hajimehoshi/ebiten)
[![GoDoc](https://godoc.org/github.com/hajimehoshi/ebiten?status.svg)](http://godoc.org/github.com/hajimehoshi/ebiten)
[![Go Report Card](https://goreportcard.com/badge/github.com/hajimehoshi/ebiten)](https://goreportcard.com/report/github.com/hajimehoshi/ebiten)

A dead simple 2D game library in Go

Website: https://hajimehoshi.github.io/ebiten/

## Platforms

* [Windows](https://github.com/hajimehoshi/ebiten/wiki/Windows)
* [macOS](https://github.com/hajimehoshi/ebiten/wiki/macOS)
* [Linux](https://github.com/hajimehoshi/ebiten/wiki/Linux)
* [FreeBSD](https://github.com/hajimehoshi/ebiten/wiki/FreeBSD)
* [Android](https://github.com/hajimehoshi/ebiten/wiki/Android)
* [iOS](https://github.com/hajimehoshi/ebiten/wiki/iOS)
* [Web browsers (Chrome, Firefox, Safari and Edge)](https://github.com/hajimehoshi/ebiten/wiki/Web-Browsers) (powered by [GopherJS](http://gopherjs.org/))

Note: Gamepad and keyboard are not available on Android/iOS.

## Features

* 2D Graphics (Geometry/Color matrix transformation, Various composition modes, Offscreen rendering, Fullscreen, Text rendering)
* Input (Mouse, Keyboard, Gamepads, Touches)
* Audio (MP3, Ogg/Vorbis, WAV, PCM, Syncing with game progress)

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
