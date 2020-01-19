# Contributing to Ebiten

Ebiten is an open source project and we would appricate your conributions!

There are some rules for Ebiten contribution.

## Asking us when you are not sure

You can ask us by [creating a GitHub issue](https://github.com/hajimehoshi/ebiten/issues/new) or at [Gophers Slack](https://invite.slack.golangbridge.org/)'s #ebiten channel.

## Following the Go convention

Please follow the Go convension like [Effective Go](https://golang.org/doc/effective_go.html).
For example, formatting by `go fmt` is required.

## Adding copyright comments to each file

```go
// Copyright [YYYY] The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
```

You don't have to update existing files' license comments.

## Adding build tags for examples

```go
// +build example jsgo
```

`example` is to prevent from installing executions by `go get github.com/hajimehoshi/ebiten/...`.

`jsgo` is to make the examples work on [jsgo.io](https://jsgo.io/).

## Documentation

See the [documents](https://ebiten.org/documents/implementation.html) about internal implementation.
