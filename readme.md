# Ebiten (海老天)

[![Build Status](https://travis-ci.org/hajimehoshi/ebiten.svg?branch=master)](https://travis-ci.org/hajimehoshi/ebiten)

* A simple SNES-like 2D game library in Go
* Works on
  * HTML5 (powered by [GopherJS](http://gopherjs.org/))
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

If you want to use GopherJS, execute this:

```
:; go get -u -tag=js github.com/hajimehoshi/ebiten
```

## Execute the example

```
:; cd $GOPATH/src/github.com/hajimehoshi/ebiten/example
:; go run blocks/main.go
```

## Execute the example on your browser

```
:; go get github.com/gopherjs/gopherjs
:; go run $GOPATH/src/github.com/hajimehoshi/ebiten/example/server/main.go
```

Then, open ``localhost:8000`` on your browser.

``localhost:8000/?EXAMPLE_NAME`` shows other examples (e.g. ``localhost:8000/?rotate``).

### Benchmark the example

```
:; cd $GOPATH/src/github.com/hajimehoshi/ebiten/example
:; go build -o=example blocks/main.go
:; ./example -cpuprofile=cpu.out
:; go tool pprof ./example cpu.out
```

## Versioning

* We obey [Semantic Versioning](http://semver.org/) basically

## License

See license.txt.
