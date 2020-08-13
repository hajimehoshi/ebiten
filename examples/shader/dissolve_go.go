// Code generated by file2byteslice. DO NOT EDIT.
// (gofmt is fine after generating)

package main

var dissolve_go = []byte("// Copyright 2020 The Ebiten Authors\n//\n// Licensed under the Apache License, Version 2.0 (the \"License\");\n// you may not use this file except in compliance with the License.\n// You may obtain a copy of the License at\n//\n//     http://www.apache.org/licenses/LICENSE-2.0\n//\n// Unless required by applicable law or agreed to in writing, software\n// distributed under the License is distributed on an \"AS IS\" BASIS,\n// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n// See the License for the specific language governing permissions and\n// limitations under the License.\n\n// +build ignore\n\npackage main\n\nvar Time float\nvar Cursor vec2\nvar ScreenImage vec2\n\nfunc Fragment(position vec4, texCoord vec2, color vec4) vec4 {\n\t// Triangle wave to go 0-->1-->0...\n\tlimit := abs(2*fract(Time/3) - 1)\n\tlevel := image3TextureAt(texCoord).x\n\n\t// Add a white border\n\tif limit-0.1 < level && level < limit {\n\t\talpha := image0TextureAt(texCoord).w\n\t\treturn vec4(alpha)\n\t}\n\n\treturn step(limit, level) * image0TextureAt(texCoord)\n}\n")
