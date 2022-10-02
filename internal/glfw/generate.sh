# Copyright 2022 The Ebitengine Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

VERSION=3.3.8

curl -L https://github.com/glfw/glfw/releases/download/$VERSION/glfw-$VERSION.zip > glfw-$VERSION.zip
unzip glfw-$VERSION.zip
cd glfw-$VERSION/
mkdir build
cd build
export MACOSX_DEPLOYMENT_TARGET=10.14
cmake -D CMAKE_BUILD_TYPE=Release -D GLFW_NATIVE_API=1 -D CMAKE_OSX_ARCHITECTURES="x86_64;arm64" -D BUILD_SHARED_LIBS=ON -D CMAKE_C_COMPILER=clang ../
make

mv src/libglfw.3.3.dylib ../../libglfw.$VERSION.dylib

cd ../../
rm glfw-$VERSION.zip
rm -r glfw-$VERSION
