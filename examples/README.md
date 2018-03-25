# How to execute the examples

```sh
go run -tags=example $GOPATH/src/github.com/hajimehoshi/ebiten/examples/rotate/main.go
```

## How to execute the examples on browsers

```sh
gopherjs serve --tags=example
```

and access `http://127.0.0.1:8080/github.com/hajimehoshi/ebiten/examples`.

## How to execute the examples on Android

Install [gomobile](https://godoc.org/golang.org/x/mobile/cmd/gomobile) first.

```sh
gomobile install -tags="gomobilebuild example" github.com/hajimehoshi/ebiten/examples/rotate
```
