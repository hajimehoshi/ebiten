# How to generate the doc

`go generate .`

# How to run HTTP server

`go run server/main.go`

# How to deploy

`git subtree push --prefix _docs/public/ origin gh-pages`

# How to update the version

1. Fix the release date in the HTML
2. Create a new branch from master branch with a version name like 1.2
3. In master branch:
  1. Update version.txt in the master branch like 1.3.0-alpha
4. In the new branch:
  1. Update version.txt like 1.2.0-rc1
  2. Generate the doc
  3. Add tag like v1.2.0-rc1
  4. Deploy JavaScript files to github.com/hajimehoshi/ebiten.pagestorage
  5. Deploy the doc (You might see confliction. Unfortunately, we might have to use git push -f (See https://gist.github.com/cobyism/4730490#gistcomment-1394421))
