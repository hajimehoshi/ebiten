# How to generate the doc

`go generate .`

# How to run HTTP server

`go run server/main.go`

# How to update the version

1. Create a new branch from master branch with a version name like 1.2
2. In master branch:
  1. Update version.txt in the master branch like 1.3.0-alpha
3. In the new branch:
  1. Update version.txt like 1.2.0-rc1
  2. Generate the doc
  3. Add tag like v1.2.0-rc1
  4. Deploy JavaScript files to github.com/hajimehoshi/ebiten.pagestorage
  5. Deploy the doc with `go generate`
