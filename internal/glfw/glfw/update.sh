set -e

dir=$(go list -f '{{.Dir}}' github.com/go-gl/glfw/v3.3/glfw)/glfw
ls ./deps/mingw | xargs -I '{}' cp $dir/deps/mingw/'{}' ./deps/mingw/'{}'
ls ./include/GLFW | xargs -I '{}' cp $dir/include/GLFW/'{}' ./include/GLFW/'{}'
ls ./src | xargs -I '{}' cp $dir/src/'{}' ./src/'{}'
echo 'Apply the change based on README.md later.'
