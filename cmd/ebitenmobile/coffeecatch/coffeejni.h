/* CoffeeCatch, a tiny native signal handler/catcher for JNI code.
 * (especially for Android/Dalvik)
 *
 * Copyright (c) 2013, Xavier Roche (http://www.httrack.com/)
 * All rights reserved.
 * See the "License" section below for the licensing terms.
 *
 * Description:
 *
 * Allows to "gracefully" recover from a signal (segv, sibus...) as if it was
 * a Java exception. It will not gracefully recover from allocator/mutexes
 * corruption etc., however, but at least "most" gentle crashes (null pointer
 * dereferencing, integer division, stack overflow etc.) should be handled
 * without too much troubles.
 *
 * The handler is thread-safe, but client must have exclusive control on the
 * signal handlers (ie. the library is installing its own signal handlers on
 * top of the existing ones).
 *
 * You must build all your libraries with `-funwind-tables', to get proper
 * unwinding information on all binaries. On ARM, you may also use the
 * `--no-merge-exidx-entries` linker switch, to solve certain issues with
 * unwinding (the switch is possibly not needed anymore).
 * On Android, this can be achieved by using this line in the Android.mk file
 * in each library block:
 *   LOCAL_CFLAGS := -funwind-tables -Wl,--no-merge-exidx-entries
 *
 * Example:
 * COFFEE_TRY_JNI(env, *retcode = call_dangerous_function(env, object));
 *
 * Implementation notes:
 *
 * Currently the library is installing both alternate stack and signal
 * handlers for known signals (SIGABRT, SIGILL, SIGTRAP, SIGBUS, SIGFPE,
 * SIGSEGV, SIGSTKFLT), and is using sigsetjmp()/siglongjmp() to return to
 * "userland" (compared to signal handler context). As a security, an alarm
 * is started as soon as a fatal signal is detected (ie. not something the
 * JVM will handle) to kill the process after a grace period. Be sure your
 * program will exit quickly after the error is caught, or call alarm(0)
 * to cancel the pending time-bomb.
 * The signal handlers had to be written with caution, because the virtual
 * machine might be using signals (including SEGV) to handle JIT compiler,
 * and some clever optimizations (such as NullPointerException handling)
 * We are using several signal-unsafe functions, namely:
 * - siglongjmp() to return to userland
 * - pthread_getspecific() to get thread-specific setup
 *
 * License:
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

#include <jni.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Setup crash handler to enter in a protected section. If a recognized signal
 * is received in this section, an appropriate native Java Error will be
 * raised.
 *
 * You can not exit the protected section block CODE_TO_BE_EXECUTED, using 
 * statements such as "return", because the cleanup code would not be
 * executed.
 *
 * It is advised to enclose the complete CODE_TO_BE_EXECUTED block in a
 * dedicated function declared extern or __attribute__ ((noinline)).
 *
 * You must build all your libraries with `-funwind-tables', to get proper
 * unwinding information on all binaries. On Android, this can be achieved
 * by using this line in the Android.mk file in each library block:
 *   LOCAL_CFLAGS := -funwind-tables
 *
 * Example:
 *
 * void my_native_function(JNIEnv* env, jobject object, jint *retcode) {
 *   COFFEE_TRY_JNI(env, *retcode = call_dangerous_function(env, object));
 * }
 *
 * In addition, the following restrictions MUST be followed:
 * - the function must be declared extern, or with the special attribute
 *   __attribute__ ((noinline)).
 * - you must not use local variables before the COFFEE_TRY_JNI block,
 *   or define them as "volatile".
 *
COFFEE_TRY_JNI(JNIEnv* env, CODE_TO_BE_EXECUTED)
 */

/** Internal functions & definitions, not to be used directly. **/
extern void coffeecatch_throw_exception(JNIEnv* env);
#define COFFEE_TRY_JNI(ENV, CODE)       \
  do {                                  \
    COFFEE_TRY() {                      \
      CODE;                             \
    } COFFEE_CATCH() {                  \
      coffeecatch_throw_exception(ENV); \
    } COFFEE_END();                     \
  } while(0)
/** End of internal functions & definitions. **/

#ifdef __cplusplus
}
#endif

#endif
