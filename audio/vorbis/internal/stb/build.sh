emcc -Os -o stbvorbis.js -g -DSTB_VORBIS_NO_STDIO -s WASM=1 -s EXPORTED_FUNCTIONS='["_stb_vorbis_decode_memory"]' -s EXTRA_EXPORTED_RUNTIME_METHODS='["ccall","cwrap"]' -s ALLOW_MEMORY_GROWTH=1 stb_vorbis.c
go run genwasmjs.go < stbvorbis.wasm > wasm.js
