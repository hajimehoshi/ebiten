// Copyright 2018 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

var _ebiten = {};

(() => {
  var decodeMemory = null;
  var vorbisDecoderInitialized = null;

  _ebiten.initializeVorbisDecoder = (callback) => {
    Module.run();
    vorbisDecoderInitialized = callback;
  };

  Module.onRuntimeInitialized = () => {
    decodeMemory = Module.cwrap('stb_vorbis_decode_memory', 'number', ['number', 'number', 'number', 'number', 'number']);
    if (vorbisDecoderInitialized) {
      vorbisDecoderInitialized();
    }
  }

  function arrayToHeap(typedArray){
    const ptr = Module._malloc(typedArray.byteLength);
    const heapBytes = new Uint8Array(Module.HEAPU8.buffer, ptr, typedArray.byteLength);
    heapBytes.set(new Uint8Array(typedArray.buffer, typedArray.byteOffset, typedArray.byteLength));
    return heapBytes;
  }

  function ptrToInt32(ptr) {
    const a = new Int32Array(Module.HEAPU8.buffer, ptr, 1);
    return a[0];
  }

  function ptrToFloat32(ptr) {
    const a = new Float32Array(Module.HEAPU8.buffer, ptr, 1);
    return a[0];
  }

  function ptrToInt16s(ptr, length) {
    const buf = new ArrayBuffer(length * Int16Array.BYTES_PER_ELEMENT);
    const copied = new Int16Array(buf);
    copied.set(new Int16Array(Module.HEAPU8.buffer, ptr, length));
    return copied;
  }

  _ebiten.decodeVorbis = (buf) => {
    const copiedBuf = arrayToHeap(buf);
    const channelsPtr = Module._malloc(4);
    const sampleRatePtr = Module._malloc(4);
    const outputPtr = Module._malloc(4);
    const length = decodeMemory(copiedBuf.byteOffset, copiedBuf.length, channelsPtr, sampleRatePtr, outputPtr);
    if (length < 0) {
      return null;
    }
    const channels = ptrToInt32(channelsPtr);
    const result = {
      data:       ptrToInt16s(ptrToInt32(outputPtr), length * channels),
      channels:   channels,
      sampleRate: ptrToInt32(sampleRatePtr),
    };

    Module._free(copiedBuf.byteOffset);
    Module._free(channelsPtr);
    Module._free(sampleRatePtr);
    Module._free(ptrToInt32(outputPtr));
    Module._free(outputPtr);
    return result;
  };
})();
