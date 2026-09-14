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

// clearException clears a pending Java exception. A failed lookup like
// FindClass or GetMethodID leaves an exception pending, and any further JNI
// call made with an exception pending is undefined.
static void clearException(JNIEnv* env) {
  if ((*env)->ExceptionCheck(env)) {
    (*env)->ExceptionClear(env);
  }
}

// The following JNI code works as this pseudo Java code:
//
//     WindowService windowService = context.getSystemService(Context.WINDOW_SERVICE);
//     Display display = windowManager.getDefaultDisplay();
//     DisplayMetrics displayMetrics = new DisplayMetrics();
//     display.getRealMetrics(displayMetrics);
//     return displayMetrics.widthPixels, displayMetrics.heightPixels, displayMetrics.density;
//
// A failure leaves the default values set at the beginning.
#cgo noescape displayInfo
#cgo nocallback displayInfo
static void displayInfo(int* width, int* height, float* scale, uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx) {
  *width = 0;
  *height = 0;
  *scale = 1;

  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;
  jobject context = (jobject)ctx;

  const char* kWindowService = "window";

  // Every local reference taken below is released at cleanup.
  jclass android_content_Context = NULL;
  jclass android_view_WindowManager = NULL;
  jclass android_view_Display = NULL;
  jclass android_util_DisplayMetrics = NULL;
  jobject android_context_Context_WINDOW_SERVICE = NULL;
  jobject windowManager = NULL;
  jobject display = NULL;
  jobject displayMetrics = NULL;

  android_content_Context = (*env)->FindClass(env, "android/content/Context");
  if (!android_content_Context) {
    goto cleanup;
  }

  android_view_WindowManager = (*env)->FindClass(env, "android/view/WindowManager");
  if (!android_view_WindowManager) {
    goto cleanup;
  }

  android_view_Display = (*env)->FindClass(env, "android/view/Display");
  if (!android_view_Display) {
    goto cleanup;
  }

  android_util_DisplayMetrics = (*env)->FindClass(env, "android/util/DisplayMetrics");
  if (!android_util_DisplayMetrics) {
    goto cleanup;
  }

  const jfieldID windowServiceID =
      (*env)->GetStaticFieldID(env, android_content_Context, "WINDOW_SERVICE", "Ljava/lang/String;");
  if (!windowServiceID) {
    goto cleanup;
  }

  const jmethodID getSystemServiceID =
      (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;");
  if (!getSystemServiceID) {
    goto cleanup;
  }

  const jmethodID getDefaultDisplayID =
      (*env)->GetMethodID(env, android_view_WindowManager, "getDefaultDisplay", "()Landroid/view/Display;");
  if (!getDefaultDisplayID) {
    goto cleanup;
  }

  const jmethodID displayMetricsInitID =
      (*env)->GetMethodID(env, android_util_DisplayMetrics, "<init>", "()V");
  if (!displayMetricsInitID) {
    goto cleanup;
  }

  const jmethodID getRealMetricsID =
      (*env)->GetMethodID(env, android_view_Display, "getRealMetrics", "(Landroid/util/DisplayMetrics;)V");
  if (!getRealMetricsID) {
    goto cleanup;
  }

  const jfieldID widthPixelsID =
      (*env)->GetFieldID(env, android_util_DisplayMetrics, "widthPixels", "I");
  if (!widthPixelsID) {
    goto cleanup;
  }

  const jfieldID heightPixelsID =
      (*env)->GetFieldID(env, android_util_DisplayMetrics, "heightPixels", "I");
  if (!heightPixelsID) {
    goto cleanup;
  }

  const jfieldID densityID =
      (*env)->GetFieldID(env, android_util_DisplayMetrics, "density", "F");
  if (!densityID) {
    goto cleanup;
  }

  android_context_Context_WINDOW_SERVICE =
      (*env)->GetStaticObjectField(env, android_content_Context, windowServiceID);

  // The result of a Java method call is invalid when the call throws, so an
  // exception is checked ahead of the result. The context here is the
  // application context, a non-visual context for which retrieving the window
  // service is discouraged since API level 30 and can return null.
  windowManager =
      (*env)->CallObjectMethod(env, context, getSystemServiceID, android_context_Context_WINDOW_SERVICE);
  if ((*env)->ExceptionCheck(env) || !windowManager) {
    goto cleanup;
  }

  display = (*env)->CallObjectMethod(env, windowManager, getDefaultDisplayID);
  if ((*env)->ExceptionCheck(env) || !display) {
    goto cleanup;
  }

  displayMetrics = (*env)->NewObject(env, android_util_DisplayMetrics, displayMetricsInitID);
  if ((*env)->ExceptionCheck(env) || !displayMetrics) {
    goto cleanup;
  }

  (*env)->CallVoidMethod(env, display, getRealMetricsID, displayMetrics);
  if ((*env)->ExceptionCheck(env)) {
    goto cleanup;
  }

  *width = (*env)->GetIntField(env, displayMetrics, widthPixelsID);
  *height = (*env)->GetIntField(env, displayMetrics, heightPixelsID);
  *scale = (*env)->GetFloatField(env, displayMetrics, densityID);

cleanup:
  clearException(env);

  // DeleteLocalRef ignores NULL on Android (ART and Dalvik), which the JNI
  // specification leaves unspecified.
  (*env)->DeleteLocalRef(env, displayMetrics);
  (*env)->DeleteLocalRef(env, display);
  (*env)->DeleteLocalRef(env, windowManager);
  (*env)->DeleteLocalRef(env, android_context_Context_WINDOW_SERVICE);
  (*env)->DeleteLocalRef(env, android_util_DisplayMetrics);
  (*env)->DeleteLocalRef(env, android_view_Display);
  (*env)->DeleteLocalRef(env, android_view_WindowManager);
  (*env)->DeleteLocalRef(env, android_content_Context);
}
*/
import "C"

import (
	"errors"

	"github.com/ebitengine/gomobile/app"

	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

type graphicsDriverCreatorImpl struct {
	colorSpace color.ColorSpace
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

// refreshDisplayInfo records the display info for displayInfo to serve on any
// thread. refreshDisplayInfo must be called on the main thread.
func (u *UserInterface) refreshDisplayInfo() {
	var cWidth, cHeight C.int
	var cScale C.float
	if err := app.RunOnJVM(func(vm, env, ctx uintptr) error {
		C.displayInfo(&cWidth, &cHeight, &cScale, C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx))
		return nil
	}); err != nil {
		// JVM is not ready yet.
		// TODO: Fix gomobile to detect the error type for this case.
		return
	}
	theDisplayInfo.Store(&displayInfoValues{
		width:  float64(cWidth),
		height: float64(cHeight),
		scale:  float64(cScale),
	})
}

func (u *UserInterface) RunOnMainThread(f func()) {
	panic("ui: RunOnMainThread is not supported for this platform")
}
