# How to generate the doc

`go generate .`

# How to run HTTP server

`go run server/main.go`

# How to update the version

0. Check all example work on all platforms
1. Create a new branch from master branch with a version name like 1.2
2. In the new branch:
  1. Update version.txt like 1.2.0
  2. Add tag like v1.2.0
3. In master branch:
  1. Update version.txt in the master branch like 1.3.0-alpha
  2. Deploy the doc with `go generate`
