# oksvg (fork)

This package is a fork of [github.com/srwiley/oksvg](https://github.com/srwiley/oksvg)
at version `v0.0.0-20221011165216-be6e8873101c`. The original is licensed
under the BSD-3-Clause license; its full text is reproduced in the License
section of the repository root README.md. This fork is distributed under
the Apache License 2.0 like the rest of this repository.

The tests, testdata, and doc directory of the original repository are not
included.

## Local modifications

- Gradient rendering under a drawing transform: `SvgPath.DrawTransformed`
  now builds gradient color functions with
  `Gradient.GetColorFunctionUS(opacity, objMatrix)`, passing the composed
  user-to-device matrix, instead of `Gradient.GetColorFunction(opacity)`,
  which ignores the transform. Without this, gradients with
  `gradientUnits="userSpaceOnUse"` were evaluated against untransformed
  geometry and rendered incorrectly when a transform was applied.
- The `golang.org/x/net/html/charset` dependency is removed:
  `ReadIconStream` no longer sets `decoder.CharsetReader`. Consumers of
  this fork feed it OpenType SVG documents, which are UTF-8 by
  specification, and encoding/xml handles UTF-8 natively.
- Mechanical modernizations applied by `go fix` (e.g. `interface{}` to
  `any`).
- The original license headers are replaced with SPDX-style headers
  crediting the original authors.
