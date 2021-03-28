// Copyright 2021 The Ebiten Authors
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

package readerdriver

// TODO: Use AAudio and OpenSL. See https://github.com/google/oboe.

/*

#cgo LDFLAGS: -llog

#include <jni.h>
#include <stdlib.h>

static jclass android_media_AudioAttributes = NULL;
static jclass android_media_AudioAttributes_Builder = NULL;
static jclass android_media_AudioFormat = NULL;
static jclass android_media_AudioFormat_Builder = NULL;
static jclass android_media_AudioManager = NULL;
static jclass android_media_AudioTrack = NULL;

static char* initAudioTrack(uintptr_t java_vm, uintptr_t jni_env,
    int sampleRate, int channelNum, int bitDepthInBytes, jobject* audioTrack, int bufferSize) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;

  jclass android_os_Build_VERSION = (*env)->FindClass(env, "android/os/Build$VERSION");
  const jint availableSDK =
      (*env)->GetStaticIntField(
          env, android_os_Build_VERSION,
          (*env)->GetStaticFieldID(env, android_os_Build_VERSION, "SDK_INT", "I"));
  (*env)->DeleteLocalRef(env, android_os_Build_VERSION);

  if (!android_media_AudioFormat) {
    jclass local = (*env)->FindClass(env, "android/media/AudioFormat");
    android_media_AudioFormat = (*env)->NewGlobalRef(env, local);
    (*env)->DeleteLocalRef(env, local);
  }

  if (!android_media_AudioManager) {
    jclass local = (*env)->FindClass(env, "android/media/AudioManager");
    android_media_AudioManager = (*env)->NewGlobalRef(env, local);
    (*env)->DeleteLocalRef(env, local);
  }

  if (!android_media_AudioTrack) {
    jclass local = (*env)->FindClass(env, "android/media/AudioTrack");
    android_media_AudioTrack = (*env)->NewGlobalRef(env, local);
    (*env)->DeleteLocalRef(env, local);
  }

  const jint android_media_AudioManager_STREAM_MUSIC =
      (*env)->GetStaticIntField(
          env, android_media_AudioManager,
          (*env)->GetStaticFieldID(env, android_media_AudioManager, "STREAM_MUSIC", "I"));
  const jint android_media_AudioTrack_MODE_STREAM =
      (*env)->GetStaticIntField(
          env, android_media_AudioTrack,
          (*env)->GetStaticFieldID(env, android_media_AudioTrack, "MODE_STREAM", "I"));
  const jint android_media_AudioFormat_CHANNEL_OUT_MONO =
      (*env)->GetStaticIntField(
          env, android_media_AudioFormat,
          (*env)->GetStaticFieldID(env, android_media_AudioFormat, "CHANNEL_OUT_MONO", "I"));
  const jint android_media_AudioFormat_CHANNEL_OUT_STEREO =
      (*env)->GetStaticIntField(
          env, android_media_AudioFormat,
          (*env)->GetStaticFieldID(env, android_media_AudioFormat, "CHANNEL_OUT_STEREO", "I"));
  const jint android_media_AudioFormat_ENCODING_PCM_8BIT =
      (*env)->GetStaticIntField(
          env, android_media_AudioFormat,
          (*env)->GetStaticFieldID(env, android_media_AudioFormat, "ENCODING_PCM_8BIT", "I"));
  const jint android_media_AudioFormat_ENCODING_PCM_16BIT =
      (*env)->GetStaticIntField(
          env, android_media_AudioFormat,
          (*env)->GetStaticFieldID(env, android_media_AudioFormat, "ENCODING_PCM_16BIT", "I"));

  jint channel = android_media_AudioFormat_CHANNEL_OUT_MONO;
  switch (channelNum) {
  case 1:
    channel = android_media_AudioFormat_CHANNEL_OUT_MONO;
    break;
  case 2:
    channel = android_media_AudioFormat_CHANNEL_OUT_STEREO;
    break;
  default:
    return "invalid channel";
  }

  jint encoding = android_media_AudioFormat_ENCODING_PCM_8BIT;
  switch (bitDepthInBytes) {
  case 1:
    encoding = android_media_AudioFormat_ENCODING_PCM_8BIT;
    break;
  case 2:
    encoding = android_media_AudioFormat_ENCODING_PCM_16BIT;
    break;
  default:
    return "invalid bitDepthInBytes";
  }

  // If the available Android SDK is at least 24 (7.0 Nougat), the FLAG_LOW_LATENCY is available.
  // This requires a different constructor.
  if (availableSDK >= 24) {
    if (!android_media_AudioAttributes_Builder) {
      jclass local = (*env)->FindClass(env, "android/media/AudioAttributes$Builder");
      android_media_AudioAttributes_Builder = (*env)->NewGlobalRef(env, local);
      (*env)->DeleteLocalRef(env, local);
    }

    if (!android_media_AudioFormat_Builder) {
      jclass local = (*env)->FindClass(env, "android/media/AudioFormat$Builder");
      android_media_AudioFormat_Builder = (*env)->NewGlobalRef(env, local);
      (*env)->DeleteLocalRef(env, local);
    }

    if (!android_media_AudioAttributes) {
      jclass local = (*env)->FindClass(env, "android/media/AudioAttributes");
      android_media_AudioAttributes = (*env)->NewGlobalRef(env, local);
      (*env)->DeleteLocalRef(env, local);
    }

    jint android_media_AudioAttributes_USAGE_UNKNOWN =
        (*env)->GetStaticIntField(
            env, android_media_AudioAttributes,
            (*env)->GetStaticFieldID(env, android_media_AudioAttributes, "USAGE_UNKNOWN", "I"));
    jint android_media_AudioAttributes_CONTENT_TYPE_UNKNOWN =
        (*env)->GetStaticIntField(
            env, android_media_AudioAttributes,
            (*env)->GetStaticFieldID(env, android_media_AudioAttributes, "CONTENT_TYPE_UNKNOWN", "I"));
    jint android_media_AudioAttributes_FLAG_LOW_LATENCY =
        (*env)->GetStaticIntField(
            env, android_media_AudioAttributes,
            (*env)->GetStaticFieldID(env, android_media_AudioAttributes, "FLAG_LOW_LATENCY", "I"));

    const jobject aattrBld =
        (*env)->NewObject(
            env, android_media_AudioAttributes_Builder,
            (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "<init>", "()V"));

    (*env)->CallObjectMethod(
        env, aattrBld,
        (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "setUsage", "(I)Landroid/media/AudioAttributes$Builder;"),
        android_media_AudioAttributes_USAGE_UNKNOWN);
    (*env)->CallObjectMethod(
        env, aattrBld,
        (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "setContentType", "(I)Landroid/media/AudioAttributes$Builder;"),
        android_media_AudioAttributes_CONTENT_TYPE_UNKNOWN);
    (*env)->CallObjectMethod(
        env, aattrBld,
        (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "setFlags", "(I)Landroid/media/AudioAttributes$Builder;"),
        android_media_AudioAttributes_FLAG_LOW_LATENCY);
    const jobject aattr =
        (*env)->CallObjectMethod(
            env, aattrBld,
            (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "build", "()Landroid/media/AudioAttributes;"));
    (*env)->DeleteLocalRef(env, aattrBld);

    const jobject afmtBld =
        (*env)->NewObject(
            env, android_media_AudioFormat_Builder,
            (*env)->GetMethodID(env, android_media_AudioFormat_Builder, "<init>", "()V"));
    (*env)->CallObjectMethod(
        env, afmtBld,
        (*env)->GetMethodID(env, android_media_AudioFormat_Builder, "setSampleRate", "(I)Landroid/media/AudioFormat$Builder;"),
        sampleRate);
    (*env)->CallObjectMethod(
        env, afmtBld,
        (*env)->GetMethodID(env, android_media_AudioFormat_Builder, "setEncoding", "(I)Landroid/media/AudioFormat$Builder;"),
        encoding);
    (*env)->CallObjectMethod(
        env, afmtBld,
        (*env)->GetMethodID(env, android_media_AudioFormat_Builder, "setChannelMask", "(I)Landroid/media/AudioFormat$Builder;"),
        channel);
    const jobject afmt =
        (*env)->CallObjectMethod(
            env, afmtBld,
            (*env)->GetMethodID(env, android_media_AudioFormat_Builder, "build", "()Landroid/media/AudioFormat;"));
    (*env)->DeleteLocalRef(env, afmtBld);

    const jobject tmpAudioTrack =
        (*env)->NewObject(
            env, android_media_AudioTrack,
            (*env)->GetMethodID(env, android_media_AudioTrack, "<init>",
                                "(Landroid/media/AudioAttributes;Landroid/media/AudioFormat;III)V"),
            aattr, afmt, bufferSize, android_media_AudioTrack_MODE_STREAM, 0);
    *audioTrack = (*env)->NewGlobalRef(env, tmpAudioTrack);
    (*env)->DeleteLocalRef(env, tmpAudioTrack);
    (*env)->DeleteLocalRef(env, aattr);
    (*env)->DeleteLocalRef(env, afmt);
  } else {
    const jobject tmpAudioTrack =
        (*env)->NewObject(
            env, android_media_AudioTrack,
            (*env)->GetMethodID(env, android_media_AudioTrack, "<init>", "(IIIIII)V"),
            android_media_AudioManager_STREAM_MUSIC,
            sampleRate, channel, encoding, bufferSize,
            android_media_AudioTrack_MODE_STREAM);
    *audioTrack = (*env)->NewGlobalRef(env, tmpAudioTrack);
    (*env)->DeleteLocalRef(env, tmpAudioTrack);
  }

  return NULL;
}

static void playAudioTrack(uintptr_t java_vm, uintptr_t jni_env,
    jobject* audioTrack) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;

  (*env)->CallVoidMethod(
      env, *audioTrack,
      (*env)->GetMethodID(env, android_media_AudioTrack, "play", "()V"));
}

static void pauseAudioTrack(uintptr_t java_vm, uintptr_t jni_env,
    jobject* audioTrack) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;

  (*env)->CallVoidMethod(
      env, *audioTrack,
      (*env)->GetMethodID(env, android_media_AudioTrack, "pause", "()V"));
}

static void flushAudioTrack(uintptr_t java_vm, uintptr_t jni_env,
    jobject* audioTrack) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;

  (*env)->CallVoidMethod(
      env, *audioTrack,
      (*env)->GetMethodID(env, android_media_AudioTrack, "flush", "()V"));
}

static char* writeToAudioTrack(uintptr_t java_vm, uintptr_t jni_env,
    jobject audioTrack, int bitDepthInBytes, void* data, int length) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;

  jbyteArray arrInBytes;
  jshortArray arrInShorts;
  switch (bitDepthInBytes) {
  case 1:
    arrInBytes = (*env)->NewByteArray(env, length);
    (*env)->SetByteArrayRegion(env, arrInBytes, 0, length, data);
    break;
  case 2:
    arrInShorts = (*env)->NewShortArray(env, length);
    (*env)->SetShortArrayRegion(env, arrInShorts, 0, length, data);
    break;
  }

  jint result;
  static jmethodID write1 = NULL;
  static jmethodID write2 = NULL;
  if (!write1) {
    write1 = (*env)->GetMethodID(env, android_media_AudioTrack, "write", "([BII)I");
  }
  if (!write2) {
    write2 = (*env)->GetMethodID(env, android_media_AudioTrack, "write", "([SII)I");
  }
  switch (bitDepthInBytes) {
  case 1:
    result = (*env)->CallIntMethod(env, audioTrack, write1, arrInBytes, 0, length);
    (*env)->DeleteLocalRef(env, arrInBytes);
    break;
  case 2:
    result = (*env)->CallIntMethod(env, audioTrack, write2, arrInShorts, 0, length);
    (*env)->DeleteLocalRef(env, arrInShorts);
    break;
  }

  switch (result) {
  case -3: // ERROR_INVALID_OPERATION
    return "invalid operation";
  case -2: // ERROR_BAD_VALUE
    return "bad value";
  case -1: // ERROR
    return "error";
  }
  if (result < 0) {
    return "unknown error";
  }
  return NULL;
}

static char* releaseAudioTrack(uintptr_t java_vm, uintptr_t jni_env,
    jobject audioTrack) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;

  (*env)->CallVoidMethod(
      env, audioTrack,
      (*env)->GetMethodID(env, android_media_AudioTrack, "release", "()V"));
  return NULL;
}

*/
import "C"

import (
	"errors"
	"io"
	"sync"
	"unsafe"

	"golang.org/x/mobile/app"
)

func IsAvailable() bool {
	return true
}

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int
}

func NewContext(sampleRate int, channelNum int, bitDepthInBytes int) (Context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)
	return &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
	}, ready, nil
}

func (c *context) NewPlayer(src io.Reader) Player {
	return &player{
		context: c,
		src:     src,
	}
}

func (c *context) Close() error {
	// TODO: Implement this
	return nil
}

type player struct {
	context     *context
	src         io.Reader
	err         error
	state       playerState
	eof         bool
	abortLoopCh chan struct{}

	audioTrack C.jobject

	m sync.Mutex
}

func (p *player) Pause() {
	p.m.Lock()
	defer p.m.Unlock()
	p.pause()
}

func (p *player) pause() {
	if p.state != playerPlay {
		return
	}
	if p.audioTrack == 0 {
		return
	}

	close(p.abortLoopCh)
	p.abortLoopCh = nil

	p.state = playerPaused
	_ = app.RunOnJVM(func(vm, env, ctx uintptr) error {
		C.pauseAudioTrack(C.uintptr_t(vm), C.uintptr_t(env), &p.audioTrack)
		return nil
	})
}

func (p *player) Play() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.state != playerPaused {
		return
	}
	if p.eof {
		p.pause()
		return
	}
	if p.audioTrack == 0 {
		if err := app.RunOnJVM(func(vm, env, ctx uintptr) error {
			var audioTrack C.jobject
			if msg := C.initAudioTrack(C.uintptr_t(vm), C.uintptr_t(env),
				C.int(p.context.sampleRate), C.int(p.context.channelNum), C.int(p.context.bitDepthInBytes),
				&audioTrack, C.int(p.context.MaxBufferSize())); msg != nil {
				return errors.New("readerdriver: initAutioTrack failed: " + C.GoString(msg))
			}
			p.audioTrack = audioTrack
			return nil
		}); err != nil {
			p.err = err
			p.close()
			return
		}
	}

	p.state = playerPlay
	_ = app.RunOnJVM(func(vm, env, ctx uintptr) error {
		C.playAudioTrack(C.uintptr_t(vm), C.uintptr_t(env), &p.audioTrack)
		return nil
	})

	p.abortLoopCh = make(chan struct{})
	go p.loop(p.abortLoopCh)
}

func (p *player) loop(abortLoopCh chan struct{}) {
	buf := make([]byte, 4096)
	for {
		if !p.appendBuffer(buf, abortLoopCh) {
			return
		}
	}
}

func (p *player) appendBuffer(buf []byte, abortLoopCh chan struct{}) bool {
	p.m.Lock()
	defer p.m.Unlock()

	select {
	case <-abortLoopCh:
		return false
	default:
	}

	if p.state != playerPlay {
		return false
	}

	n, err := p.src.Read(buf)
	if err != nil && err != io.EOF {
		p.err = err
		p.close()
		return false
	}

	bufInBytes := buf[:n]

	var bufInShorts []int16 // TODO: Avoid allocating
	if p.context.bitDepthInBytes == 2 {
		bufInShorts = make([]int16, len(bufInBytes)/2)
		for i := 0; i < len(bufInShorts); i++ {
			bufInShorts[i] = int16(bufInBytes[2*i]) | (int16(bufInBytes[2*i+1]) << 8)
		}
	}

	if err := app.RunOnJVM(func(vm, env, ctx uintptr) error {
		var msg *C.char
		switch p.context.bitDepthInBytes {
		case 1:
			msg = C.writeToAudioTrack(C.uintptr_t(vm), C.uintptr_t(env),
				p.audioTrack, C.int(p.context.bitDepthInBytes),
				unsafe.Pointer(&bufInBytes[0]), C.int(len(bufInBytes)))
		case 2:
			msg = C.writeToAudioTrack(C.uintptr_t(vm), C.uintptr_t(env),
				p.audioTrack, C.int(p.context.bitDepthInBytes),
				unsafe.Pointer(&bufInShorts[0]), C.int(len(bufInShorts)))
		default:
			panic("not reached")
		}
		if msg != nil {
			return errors.New("readerdriver: writeToAudioTrack failed: " + C.GoString(msg))
		}
		return nil
	}); err != nil {
		p.err = err
		p.close()
		return false
	}

	if err == io.EOF {
		p.eof = true
		p.pause()
		return false
	}

	return true
}

func (p *player) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return p.state == playerPlay
}

func (p *player) Reset() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.state == playerClosed {
		return
	}
	if p.audioTrack == 0 {
		return
	}

	p.pause()
	p.eof = false
	_ = app.RunOnJVM(func(vm, env, ctx uintptr) error {
		C.flushAudioTrack(C.uintptr_t(vm), C.uintptr_t(env), &p.audioTrack)
		return nil
	})
}

func (p *player) Volume() float64 {
	// TODO: Implement this
	return 0
}

func (p *player) SetVolume(volume float64) {
	// TODO: Implement this
}

func (p *player) UnplayedBufferSize() int64 {
	p.m.Lock()
	defer p.m.Unlock()

	// TODO: Implement this

	if p.audioTrack == 0 {
		return 0
	}
	return 0
}

func (p *player) Err() error {
	p.m.Lock()
	defer p.m.Unlock()
	return p.err
}

func (p *player) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	return p.close()
}

func (p *player) close() error {
	if p.audioTrack == 0 {
		return nil
	}

	p.state = playerClosed
	err := app.RunOnJVM(func(vm, env, ctx uintptr) error {
		if msg := C.releaseAudioTrack(C.uintptr_t(vm), C.uintptr_t(env), p.audioTrack); msg != nil {
			return errors.New("readerplayer: releaseAudioTrack failed: " + C.GoString(msg))
		}
		return nil
	})
	p.audioTrack = 0
	return err
}
