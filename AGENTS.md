# Skills

When writing or modifying Ebitengine code or applications, consult the [`skills`](skills) directory, which holds skills for working with this repository.

# Issues and pull requests

- Before filing an issue, read and follow the applicable template in [`.github/ISSUE_TEMPLATE`](.github/ISSUE_TEMPLATE).
- Before sending a pull request, read and follow [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md).
- Preserve the template's structure and fill in every required field. When submitting an issue form through a CLI or API, render each field label as a `###` heading and preserve checkbox lists.

# Test execution

- On Linux, when running Ebitengine unit tests that exercise graphics-backed APIs or use `internal/testing.MainWithRunLoop`, always run them under `xvfb-run` so Ebitengine windows cannot appear on or steal focus from the user's display.
- Prefer commands such as `xvfb-run -a go test ...` or `xvfb-run -a go test -run ... ./path`.
- Do not fall back to a bare `go test` if `xvfb-run` is unavailable; report the missing dependency instead.
- This rule does not apply to non-graphics tests or browser/WASM tests that use their own test environment.
