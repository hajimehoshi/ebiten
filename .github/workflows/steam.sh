export PATH=$PATH:/usr/local/go/bin
export CGO_CFLAGS=-std=gnu99
export DISPLAY=:99.0

# Install Go
curl --location --remote-name https://golang.org/dl/${GO_FILENAME}
rm -rf /usr/local/go && tar -C /usr/local -xzf ${GO_FILENAME}

# Run X
Xvfb :99 -screen 0 1024x768x24 > /dev/null 2>&1 &

# Run the tests
env GITHUB_ACTIONS=true go test -v ./...
