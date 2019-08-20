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
 *
 * COFFEE_TRY() {
 *   call_some_native_function()
 * } COFFEE_CATCH() {
 *   const char*const message = coffeecatch_get_message();
 *   jclass cls = (*env)->FindClass(env, "java/lang/RuntimeException");
 *   (*env)->ThrowNew(env, cls, strdup(message));
 * } COFFEE_END();
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

#ifndef COFFEECATCH_H
#define COFFEECATCH_H

#include <stdint.h>
#include <sys/types.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Setup crash handler to enter in a protected section. If a recognized signal
 * is received in this section, the execution will be diverted to the
 * COFFEE_CATCH() block.
 *
 * Note: you MUST use the following pattern when using this macro:
 * COFFEE_TRY() {
 *   .. protected section without exit point
 * } COFFEE_CATCH() {
 *   .. handler section without exit point
 * } COFFEE_END();
 *
 * You can not exit the protected section block, or the handler section block,
 * using statements such as "return", because the cleanup code would not be
 * executed.
 *
 * It is advised to enclose this complete try/catch/end block in a dedicated
 * function declared extern or __attribute__ ((noinline)).
 *
 * Example:
 *
 * void my_native_function(JNIEnv* env, jobject object, jint *retcode) {
 *   COFFEE_TRY() {
 *     *retcode = call_dangerous_function(env, object);
 *   } COFFEE_CATCH() {
 *     const char*const message = coffeecatch_get_message();
 *     jclass cls = (*env)->FindClass(env, "java/lang/RuntimeException");
 *     (*env)->ThrowNew(env, cls, strdup(message));
 *     *retcode = -1;
 *   } COFFEE_END();
 * }
 *
 * In addition, the following restrictions MUST be followed:
 * - the function must be declared extern, or with the special attribute
 *   __attribute__ ((noinline)).
 * - you must not use local variables before the complete try/catch/end block,
 *   or define them as "volatile".
 * - your function should not ignore the crash silently, as the library will
 *   ensure the process is killed after a grace period (typically 30s) to
 *   prevent any deadlock that may occur if the crash was caught inside a
 *   non-signal-safe function, for example (such as malloc()).
 *
COFFEE_TRY()
 **/

/**
 * Declare the signal handler block. This block will be executed if a signal
 * was received, and recognized, in the previous COFFEE_TRY() {} section.
 * You may call audit functions in this block, such as coffeecatch_get_signal()
 * or coffeecatch_get_message().
 *
COFFEE_CATCH()
 **/

/**
 * Declare the end of the COFFEE_TRY()/COFFEE_CATCH() section.
 * Diagnostic functions must not be called beyond this point.
 *
COFFEE_END()
 **/

/**
 * Get the signal associated with the crash.
 * This function can only be called inside a COFFEE_CATCH() block.
 */
extern int coffeecatch_get_signal(void);

/**
 * Get the full error message associated with the crash.
 * This function can only be called inside a COFFEE_CATCH() block, and the
 * returned pointer is only valid within this block. (you may want to copy
 * the string in a static buffer, or use strdup())
 */
const char* coffeecatch_get_message(void);

/**
 * Raise an abort() signal in the current thread. If the current code section
 * is protected, the 'exp', 'file' and 'line' information are stored for
 * further audit.
 */
extern void coffeecatch_abort(const char* exp, const char* file, int line);

/**
 * Assertion check. If the expression is false, an abort() signal is raised
 * using coffeecatch_abort().
 */
#define coffeecatch_assert(EXP) (void)( (EXP) || (coffeecatch_abort(#EXP, __FILE__, __LINE__), 0) )

/**
 * Get the backtrace size, or 0 upon error.
 * This function can only be called inside a COFFEE_CATCH() block.
 */
extern size_t coffeecatch_get_backtrace_size(void);

/**
 * Get the backtrace pointer, or 0 upon error.
 * This function can only be called inside a COFFEE_CATCH() block.
 */
extern uintptr_t coffeecatch_get_backtrace(ssize_t index);

/**
 * Enumerate the backtrace with information.
 * This function can only be called inside a COFFEE_CATCH() block.
 */
extern void coffeecatch_get_backtrace_info(void (*fun)(void *arg,
                                           const char *module,
                                           uintptr_t addr,
                                           const char *function,
                                           uintptr_t offset), void *arg);

/**
 * Cancel any pending alarm() triggered after a signal was caught.
 * Calling this function is dangerous, because it exposes the process to
 * a possible deadlock if the signal was caught due to internal low-level
 * library error (mutex being in a locked state, for example).
 */
extern int coffeecatch_cancel_pending_alarm(void);

/** Internal functions & definitions, not to be used directly. **/
#include <setjmp.h>
extern int coffeecatch_inside(void);
extern int coffeecatch_setup(void);
extern sigjmp_buf* coffeecatch_get_ctx(void);
extern void coffeecatch_cleanup(void);
#define COFFEE_TRY()                                \
  if (coffeecatch_inside() || \
      (coffeecatch_setup() == 0 \
       && sigsetjmp(*coffeecatch_get_ctx(), 1) == 0))
#define COFFEE_CATCH() else
#define COFFEE_END() coffeecatch_cleanup()
/** End of internal functions & definitions. **/

#ifdef __cplusplus
}
#endif

#endif

