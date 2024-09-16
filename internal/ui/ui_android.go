// Copyright 2022 The Ebiten Authors
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

package ui

/*
#include <jni.h>
#include <stdlib.h>

// The following JNI code works as this pseudo Java code:
//
//     WindowService windowService = context.getSystemService(Context.WINDOW_SERVICE);
//     Display display = windowManager.getDefaultDisplay();
//     DisplayMetrics displayMetrics = new DisplayMetrics();
//     display.getRealMetrics(displayMetrics);
//     return displayMetrics.widthPixels, displayMetrics.heightPixels, displayMetrics.density;
//
static void displayInfo(int* width, int* height, float* scale, uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx) {
  *width = 0;
  *height = 0;
  *scale = 1;

  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;
  jobject context = (jobject)ctx;

  const char* kWindowService = "window";

  const jclass android_content_Context =
      (*env)->FindClass(env, "android/content/Context");
  const jclass android_view_WindowManager =
      (*env)->FindClass(env, "android/view/WindowManager");
  const jclass android_view_Display =
      (*env)->FindClass(env, "android/view/Display");
  const jclass android_util_DisplayMetrics =
      (*env)->FindClass(env, "android/util/DisplayMetrics");

  const jobject android_context_Context_WINDOW_SERVICE =
      (*env)->GetStaticObjectField(
          env, android_content_Context,
          (*env)->GetStaticFieldID(env, android_content_Context, "WINDOW_SERVICE", "Ljava/lang/String;"));

  const jobject windowManager =
      (*env)->CallObjectMethod(
          env, context,
          (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;"),
          android_context_Context_WINDOW_SERVICE);
  const jobject display =
      (*env)->CallObjectMethod(
          env, windowManager,
          (*env)->GetMethodID(env, android_view_WindowManager, "getDefaultDisplay", "()Landroid/view/Display;"));
  const jobject displayMetrics =
      (*env)->NewObject(
          env, android_util_DisplayMetrics,
          (*env)->GetMethodID(env, android_util_DisplayMetrics, "<init>", "()V"));
  (*env)->CallVoidMethod(
      env, display,
      (*env)->GetMethodID(env, android_view_Display, "getRealMetrics", "(Landroid/util/DisplayMetrics;)V"),
      displayMetrics);
  *width =
      (*env)->GetIntField(
          env, displayMetrics,
          (*env)->GetFieldID(env, android_util_DisplayMetrics, "widthPixels", "I"));
  *height =
      (*env)->GetIntField(
          env, displayMetrics,
          (*env)->GetFieldID(env, android_util_DisplayMetrics, "heightPixels", "I"));
  *scale =
      (*env)->GetFloatField(
          env, displayMetrics,
          (*env)->GetFieldID(env, android_util_DisplayMetrics, "density", "F"));

  (*env)->DeleteLocalRef(env, android_content_Context);
  (*env)->DeleteLocalRef(env, android_view_WindowManager);
  (*env)->DeleteLocalRef(env, android_view_Display);
  (*env)->DeleteLocalRef(env, android_util_DisplayMetrics);

  (*env)->DeleteLocalRef(env, android_context_Context_WINDOW_SERVICE);
  (*env)->DeleteLocalRef(env, windowManager);
  (*env)->DeleteLocalRef(env, display);
  (*env)->DeleteLocalRef(env, displayMetrics);
}
*/
import "C"

import (
	"errors"

	"github.com/ebitengine/gomobile/app"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverCreatorImpl struct {
	colorSpace graphicsdriver.ColorSpace
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	graphics, err := g.newOpenGL()
	return graphics, GraphicsLibraryOpenGL, err
}

func (g *graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: DirectX is not supported in this environment")
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: Metal is not supported in this environment")
}

func (*graphicsDriverCreatorImpl) newPlayStation5() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: PlayStation 5 is not supported in this environment")
}

func dipToNativePixels(x float64, scale float64) float64 {
	return x * scale
}

func dipFromNativePixels(x float64, scale float64) float64 {
	return x / scale
}

func (u *UserInterface) displayInfo() (int, int, float64, bool) {
	var cWidth, cHeight C.int
	var cScale C.float
	if err := app.RunOnJVM(func(vm, env, ctx uintptr) error {
		C.displayInfo(&cWidth, &cHeight, &cScale, C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx))
		return nil
	}); err != nil {
		// JVM is not ready yet.
		// TODO: Fix gomobile to detect the error type for this case.
		return 0, 0, 1, false
	}
	scale := float64(cScale)
	width := int(dipFromNativePixels(float64(cWidth), scale))
	height := int(dipFromNativePixels(float64(cHeight), scale))
	return width, height, scale, true
}
