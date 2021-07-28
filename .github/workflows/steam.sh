export PATH=$PATH:/usr/local/go/bin
export CGO_CFLAGS=-std=gnu99
export DISPLAY=:99.0

# Install Go
curl -L --output go${1}.linux-amd64.tar.gz https://golang.org/dl/go${1}.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go${1}.linux-amd64.tar.gz

# Run X
Xvfb :99 -screen 0 1024x768x24 > /dev/null 2>&1 &

# Run the tests
go test -tags=example -v ./...
