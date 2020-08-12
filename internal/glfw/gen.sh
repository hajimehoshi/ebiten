docker run --rm --volume $(pwd)/../..:/work $(docker build -q .) /bin/bash -c "cd ./internal/glfw; go run gen.go"
