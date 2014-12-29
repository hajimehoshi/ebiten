# Ebiten (海老天)

[![Build Status](https://travis-ci.org/hajimehoshi/ebiten.svg?branch=master)](https://travis-ci.org/hajimehoshi/ebiten)

* A simple SNES-like 2D game library in Go
* Works on
  * Mac OS X
  * Linux (maybe)
  * Windows (possibly)
* [API Docs](http://godoc.org/github.com/hajimehoshi/ebiten)

## Features

* 2D Graphics
* Input (Mouse, Keyboard)

## Example

* example/blocks - Puzzle game you know
* example/hue - Changes the hue of an image
* example/mosaic - Mosaics an image
* example/perspective - See an image in a perspective view
* example/rotate - Rotates an image
* etc.

## Install on Mac OS X

```
:; brew install glew
:; brew install glfw3 # or homebrew/versions/glfw3
:; go get -u github.com/hajimehoshi/ebiten
```

## Execute the example

```
:; cd $GOHOME/src/github.com/hajimehoshi/ebiten/example
:; go run rotate/main.go
```

### Benchmark the example

```
:; cd $GOHOME/src/github.com/hajimehoshi/ebiten/example
:; go build -o=example blocks/main.go
:; ./example -cpuprofile=cpu.out
:; go tool pprof ./example cpu.out
```

## Versioning

* We adopted [Semantic Versioning](http://semver.org/)

## License

See license.txt.
