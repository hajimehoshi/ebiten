// Copyright 2015 The Go Authors. All rights reserved.

// This file is copied from golang.org/x/mobile/internal/mobileinit/ctx_android.go
// and editted.
// This file is licensed under the 3-clause BSD license.

package jni

/*
#include <jni.h>
#include <stdlib.h>

// These definitions are duplicated with those in ctx_android.go of golang.org/x/mobile/internal/mobileinit package.
// To be exact, this might cause undefined behavior, but some compilers including GCC and Clang work as a common extension.
// (J.5.11 Multiple external definitions)
JavaVM* current_vm;
jobject current_ctx;

static char* lockJNI(uintptr_t* envp, int* attachedp) {
	JNIEnv* env;

	if (current_vm == NULL) {
		return "no current JVM";
	}

	*attachedp = 0;
	switch ((*current_vm)->GetEnv(current_vm, (void**)&env, JNI_VERSION_1_6)) {
	case JNI_OK:
		break;
	case JNI_EDETACHED:
		if ((*current_vm)->AttachCurrentThread(current_vm, &env, 0) != 0) {
			return "cannot attach to JVM";
		}
		*attachedp = 1;
		break;
	case JNI_EVERSION:
		return "bad JNI version";
	default:
		return "unknown JNI error from GetEnv";
	}

	*envp = (uintptr_t)env;
	return NULL;
}

static char* checkException(uintptr_t jnienv) {
	jthrowable exc;
	JNIEnv* env = (JNIEnv*)jnienv;

	if (!(*env)->ExceptionCheck(env)) {
		return NULL;
	}

	exc = (*env)->ExceptionOccurred(env);
	(*env)->ExceptionClear(env);

	jclass clazz = (*env)->FindClass(env, "java/lang/Throwable");
	jmethodID toString = (*env)->GetMethodID(env, clazz, "toString", "()Ljava/lang/String;");
	jobject msgStr = (*env)->CallObjectMethod(env, exc, toString);
	return (char*)(*env)->GetStringUTFChars(env, msgStr, 0);
}

static void unlockJNI() {
	(*current_vm)->DetachCurrentThread(current_vm);
}
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

// RunOnJVM executes fn on the current VM context.
//
// RunOnJVM should not be called on init function since the current VM might not be initialized yet.
func RunOnJVM(fn func(vm, env, ctx uintptr) error) error {
	errch := make(chan error)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		env := C.uintptr_t(0)
		attached := C.int(0)
		if errStr := C.lockJNI(&env, &attached); errStr != nil {
			errch <- errors.New(C.GoString(errStr))
			return
		}
		if attached != 0 {
			defer C.unlockJNI()
		}

		vm := uintptr(unsafe.Pointer(C.current_vm))
		if err := fn(vm, uintptr(env), uintptr(C.current_ctx)); err != nil {
			errch <- err
			return
		}

		if exc := C.checkException(env); exc != nil {
			errch <- errors.New(C.GoString(exc))
			C.free(unsafe.Pointer(exc))
			return
		}
		errch <- nil
	}()
	return <-errch
}
