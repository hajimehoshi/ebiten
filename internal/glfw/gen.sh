docker run --rm --volume $(pwd)/../..:/work $(docker build -q . | head -n1) /bin/sh -c "cd ./internal/glfw; go run gen.go"
