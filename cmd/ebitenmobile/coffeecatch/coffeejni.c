/* CoffeeCatch, a tiny native signal handler/catcher for JNI code.
 * (especially for Android/Dalvik)
 *
 * Copyright (c) 2013, Xavier Roche (http://www.httrack.com/)
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice, this
 *    list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 * ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

#ifndef COFFEECATCH_JNI_H
#define COFFEECATCH_JNI_H

#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <sys/types.h>
#include <jni.h>
#include <assert.h>
#include "coffeecatch.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct t_bt_fun {
  JNIEnv* env;
  jclass cls;
  jclass cls_ste;
  jmethodID cons_ste;
  jobjectArray elements;
  size_t size;
  size_t index;
} t_bt_fun;

static char* bt_print(const char *function, uintptr_t offset) {
  if (function != NULL) {
    char buffer[256];
    snprintf(buffer, sizeof(buffer), "%s:%p", function, (void*) offset);
    return strdup(buffer);
  } else {
    return "<unknown>";
  }
}

static char* bt_addr(uintptr_t addr) {
  char buffer[32];
  snprintf(buffer, sizeof(buffer), "%p", (void*) addr);
  return strdup(buffer);
}

#define IS_VALID_CLASS_CHAR(C) ( \
  ((C) >= 'a' && (C) <= 'z')     \
  || ((C) >= 'A' && (C) <= 'Z')  \
  || ((C) >= '0' && (C) <= '9')  \
  || (C) == '_'                  \
  )

static char* bt_module(const char *module) {
  if (module != NULL) {
    size_t i;
    char *copy;
    if (*module == '/') {
      module++;
    }
    copy = strdup(module);
    /* Pseudo-java-class. */
    for(i = 0; copy[i] != '\0'; i++) {
      if (copy[i] == '/') {
        copy[i] = '.';
      } else if (!IS_VALID_CLASS_CHAR(copy[i])) {
        copy[i] = '_';
      }
    }
    return copy;
  } else {
    return "<unknown>";
  }
}

static void bt_fun(void *arg, const char *module, uintptr_t addr, 
                   const char *function, uintptr_t offset) {
  t_bt_fun *const t = (t_bt_fun*) arg;
  JNIEnv*const env = t->env;
  jstring declaringClass = (*env)->NewStringUTF(env, bt_module(module));
  jstring methodName = (*env)->NewStringUTF(env, bt_addr(addr));
  jstring fileName = (*env)->NewStringUTF(env, bt_print(function, offset));
  const int lineNumber = function != NULL ? 0 : -2;  /* "-2" is "inside JNI code" */
  jobject trace = (*env)->NewObject(env, t->cls_ste, t->cons_ste, 
                                    declaringClass, methodName, fileName,
                                    lineNumber);
  if (t->index < t->size) {
    (*t->env)->SetObjectArrayElement(t->env, t->elements, t->index++, trace);
  }
}

void coffeecatch_throw_exception(JNIEnv* env) {
  jclass cls = (*env)->FindClass(env, "java/lang/Error");
  jclass cls_ste = (*env)->FindClass(env, "java/lang/StackTraceElement");

  jmethodID cons = (*env)->GetMethodID(env, cls, "<init>", "(Ljava/lang/String;)V");
  jmethodID cons_cause = (*env)->GetMethodID(env, cls, "<init>", "(Ljava/lang/String;Ljava/lang/Throwable;)V");
  jmethodID cons_ste = (*env)->GetMethodID(env, cls_ste, "<init>",
    "(Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;I)V");
  jmethodID meth_sste = (*env)->GetMethodID(env, cls, "setStackTrace",
    "([Ljava/lang/StackTraceElement;)V");

  /* Exception message. */
  const char*const message = coffeecatch_get_message();
  jstring str = (*env)->NewStringUTF(env, strdup(message));

  /* Final exception. */
  jthrowable exception;

  /* Add pseudo-stack trace. */
  const ssize_t bt_size = coffeecatch_get_backtrace_size();

  assert(cls != NULL);
  assert(cls_ste != NULL);
  assert(cons != NULL);
  assert(cons_cause != NULL);
  assert(cons_ste != NULL);
  assert(meth_sste != NULL);

  assert(message != NULL);
  assert(str != NULL);

  /* Can we produce a stack trace ? */
  if (bt_size > 0) {
    /* Create secondary exception. */
    jthrowable cause = (jthrowable) (*env)->NewObject(env, cls, cons, str);

    /* Stack trace. */
    jobjectArray elements =
      (*env)->NewObjectArray(env, bt_size, cls_ste, NULL);
    if (elements != NULL) {
      t_bt_fun t;
      t.env = env;
      t.cls = cls;
      t.cls_ste = cls_ste;
      t.cons_ste = cons_ste;
      t.elements = elements;
      t.index = 0;
      t.size = bt_size;
      coffeecatch_get_backtrace_info(bt_fun, &t);
      (*env)->CallVoidMethod(env, cause, meth_sste, elements);
    }

    /* Primary exception */
    exception = (jthrowable) (*env)->NewObject(env, cls, cons_cause, str, cause);
  } else {
    /* Simple exception */
    exception = (jthrowable) (*env)->NewObject(env, cls, cons, str);
  }

  /* Throw exception. */
  if (exception != NULL) {
    (*env)->Throw(env, exception);
  } else {
    (*env)->ThrowNew(env, cls, strdup(message));
  }
}

#ifdef __cplusplus
}
#endif

#endif
