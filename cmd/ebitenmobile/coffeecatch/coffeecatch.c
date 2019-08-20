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

#ifdef __ANDROID__
#define USE_UNWIND
#define USE_CORKSCREW
#define USE_LIBUNWIND
#endif

/* #undef NO_USE_SIGALTSTACK */
/* #undef USE_SILENT_SIGALTSTACK */

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>
#include <string.h>
#include <errno.h>
#include <sys/types.h>
#include <assert.h>
#include <signal.h>
#include <setjmp.h>
#if defined(__ANDROID__) && !defined(__BIONIC_HAVE_UCONTEXT_T) && \
    defined(__arm__) && !defined(__BIONIC_HAVE_STRUCT_SIGCONTEXT)
#include <asm/sigcontext.h>
#endif 
#if (defined(USE_UNWIND) && !defined(USE_CORKSCREW))
#include <unwind.h>
#endif
#include <pthread.h>
#include <dlfcn.h>
#include "coffeecatch.h"

/*#define NDK_DEBUG 1*/
#if ( defined(NDK_DEBUG) && ( NDK_DEBUG == 1 ) )
#define DEBUG(A) do { A; } while(0)
#define FD_ERRNO 2
static void print(const char *const s) {
  size_t count;
  for(count = 0; s[count] != '\0'; count++) ;
  /* write() is async-signal-safe. */
  (void) write(FD_ERRNO, s, count);
}
#else
#define DEBUG(A)
#endif

/* Alternative stack size. */
#define SIG_STACK_BUFFER_SIZE SIGSTKSZ

#ifdef USE_UNWIND
/* Number of backtraces to get. */
#define BACKTRACE_FRAMES_MAX 32
#endif

/* Signals to be caught. */
#define SIG_CATCH_COUNT 7
static const int native_sig_catch[SIG_CATCH_COUNT + 1]
  = { SIGABRT, SIGILL, SIGTRAP, SIGBUS, SIGFPE, SIGSEGV
#ifdef SIGSTKFLT
    , SIGSTKFLT
#endif
    , 0 };

/* Maximum value of a caught signal. */
#define SIG_NUMBER_MAX 32

#if defined(__ANDROID__)
#ifndef ucontext_h_seen
#define ucontext_h_seen

/* stack_t definition */
#include <asm/signal.h>

#if defined(__arm__)

/* Taken from richard.quirk's header file. (Android does not have it) */

#if !defined(__BIONIC_HAVE_UCONTEXT_T)
typedef struct ucontext {
  unsigned long uc_flags;
  struct ucontext *uc_link;
  stack_t uc_stack;
  struct sigcontext uc_mcontext;
  unsigned long uc_sigmask;
} ucontext_t;
#endif

#elif defined(__aarch64__)

#elif defined(__i386__)

/* Taken from Google Breakpad. */

/* 80-bit floating-point register */
/*struct _libc_fpreg {
  unsigned short significand[4];
  unsigned short exponent;
};*/

/* Simple floating-point state, see FNSTENV instruction */
/*struct _libc_fpstate {
  unsigned long cw;
  unsigned long sw;
  unsigned long tag;
  unsigned long ipoff;
  unsigned long cssel;
  unsigned long dataoff;
  unsigned long datasel;
  struct _libc_fpreg _st[8];
  unsigned long status;
};*/

//typedef uint32_t  greg_t;

/*typedef struct {
  uint32_t gregs[19];
  struct _libc_fpstate* fpregs;
  uint32_t oldmask;
  uint32_t cr2;
} mcontext_t;*/

/*enum {
  REG_GS = 0,
  REG_FS,
  REG_ES,
  REG_DS,
  REG_EDI,
  REG_ESI,
  REG_EBP,
  REG_ESP,
  REG_EBX,
  REG_EDX,
  REG_ECX,
  REG_EAX,
  REG_TRAPNO,
  REG_ERR,
  REG_EIP,
  REG_CS,
  REG_EFL,
  REG_UESP,
  REG_SS,
};*/

#if !defined(__BIONIC_HAVE_UCONTEXT_T)
typedef struct ucontext {
  uint32_t uc_flags;
  struct ucontext* uc_link;
  stack_t uc_stack;
  mcontext_t uc_mcontext;
} ucontext_t;
#endif

#elif defined(__mips__)

/* Taken from Google Breakpad. */

typedef struct {
  uint32_t regmask;
  uint32_t status;
  uint64_t pc;
  uint64_t gregs[32];
  uint64_t fpregs[32];
  uint32_t acx;
  uint32_t fpc_csr;
  uint32_t fpc_eir;
  uint32_t used_math;
  uint32_t dsp;
  uint64_t mdhi;
  uint64_t mdlo;
  uint32_t hi1;
  uint32_t lo1;
  uint32_t hi2;
  uint32_t lo2;
  uint32_t hi3;
  uint32_t lo3;
} mcontext_t;

#if !defined(__BIONIC_HAVE_UCONTEXT_T)
typedef struct ucontext {
  uint32_t uc_flags;
  struct ucontext* uc_link;
  stack_t uc_stack;
  mcontext_t uc_mcontext;
} ucontext_t;
#endif

#else
#error "Architecture is not supported (unknown ucontext layout)"
#endif

#endif

#ifdef USE_CORKSCREW
typedef struct map_info_t map_info_t;
/* Extracted from Android's include/corkscrew/backtrace.h */
typedef struct {
    uintptr_t absolute_pc;
    uintptr_t stack_top;
    size_t stack_size;
} backtrace_frame_t;
typedef struct {
    uintptr_t relative_pc;
    uintptr_t relative_symbol_addr;
    char* map_name;
    char* symbol_name;
    char* demangled_name;
} backtrace_symbol_t;
/* Extracted from Android's libcorkscrew/arch-arm/backtrace-arm.c */
typedef ssize_t (*t_unwind_backtrace_signal_arch)
(siginfo_t* si, void* sc, const map_info_t* lst, backtrace_frame_t* bt,
size_t ignore_depth, size_t max_depth);
typedef map_info_t* (*t_acquire_my_map_info_list)();
typedef void (*t_release_my_map_info_list)(map_info_t* milist);
typedef void (*t_get_backtrace_symbols)(const backtrace_frame_t* backtrace,
                                        size_t frames,
                                        backtrace_symbol_t* symbols);
typedef void (*t_free_backtrace_symbols)(backtrace_symbol_t* symbols,
                                         size_t frames);
#endif

#endif

/* Process-wide crash handler structure. */
typedef struct native_code_global_struct {
  /* Initialized. */
  int initialized;

  /* Lock. */
  pthread_mutex_t mutex;

  /* Backup of sigaction. */
  struct sigaction *sa_old;
} native_code_global_struct;
#define NATIVE_CODE_GLOBAL_INITIALIZER { 0, PTHREAD_MUTEX_INITIALIZER, NULL }

/* Thread-specific crash handler structure. */
typedef struct native_code_handler_struct {
  /* Restore point context. */
  sigjmp_buf ctx;
  int ctx_is_set;
  int reenter;

  /* Alternate stack. */
  char *stack_buffer;
  size_t stack_buffer_size;
  stack_t stack_old;

  /* Signal code and info. */
  int code;
  siginfo_t si;
  ucontext_t uc;

  /* Uwind context. */
#if (defined(USE_CORKSCREW))
  backtrace_frame_t frames[BACKTRACE_FRAMES_MAX];
#elif (defined(USE_UNWIND))
  uintptr_t frames[BACKTRACE_FRAMES_MAX];
#endif
#ifdef USE_LIBUNWIND
  void* uframes[BACKTRACE_FRAMES_MAX];
#endif
  size_t frames_size;
  size_t frames_skip;

  /* Custom assertion failures. */
  const char *expression;
  const char *file;
  int line;

  /* Alarm was fired. */
  int alarm;
} native_code_handler_struct;

/* Global crash handler structure. */
static native_code_global_struct native_code_g =
  NATIVE_CODE_GLOBAL_INITIALIZER;

/* Thread variable holding context. */
pthread_key_t native_code_thread;

#if (defined(USE_UNWIND) && !defined(USE_CORKSCREW))
/* Unwind callback */
static _Unwind_Reason_Code
coffeecatch_unwind_callback(struct _Unwind_Context* context, void* arg) {
  native_code_handler_struct *const s = (native_code_handler_struct*) arg;

  const uintptr_t ip = _Unwind_GetIP(context);

  DEBUG(print("called unwind callback\n"));

  if (ip != 0x0) {
    if (s->frames_skip == 0) {
      s->frames[s->frames_size] = ip;
      s->frames_size++;
    } else {
      s->frames_skip--;
    }
  }

  if (s->frames_size == BACKTRACE_FRAMES_MAX) {
    return _URC_END_OF_STACK;
  } else {
    DEBUG(print("returned _URC_OK\n"));
    return _URC_OK;
  }
}
#endif

/* Use libcorkscrew to get a backtrace inside a signal handler.
   Will only return a non-zero code on Android >= 4 (with libcorkscrew.so
   being shipped) */
#ifdef USE_CORKSCREW
static size_t coffeecatch_backtrace_signal(siginfo_t* si, void* sc, 
                                           backtrace_frame_t* frames,
                                           size_t ignore_depth,
                                           size_t max_depth) {
  void *const libcorkscrew = dlopen("libcorkscrew.so", RTLD_LAZY | RTLD_LOCAL);
  if (libcorkscrew != NULL) {
    t_unwind_backtrace_signal_arch unwind_backtrace_signal_arch 
      = (t_unwind_backtrace_signal_arch)
      dlsym(libcorkscrew, "unwind_backtrace_signal_arch");
    t_acquire_my_map_info_list acquire_my_map_info_list 
      = (t_acquire_my_map_info_list)
      dlsym(libcorkscrew, "acquire_my_map_info_list");
    t_release_my_map_info_list release_my_map_info_list 
      = (t_release_my_map_info_list)
      dlsym(libcorkscrew, "release_my_map_info_list");
    if (unwind_backtrace_signal_arch != NULL
        && acquire_my_map_info_list != NULL
        && release_my_map_info_list != NULL) {
      map_info_t*const info = acquire_my_map_info_list();
      const ssize_t size = 
        unwind_backtrace_signal_arch(si, sc, info, frames, ignore_depth,
                                     max_depth);
      release_my_map_info_list(info);
      return size >= 0 ? size : 0;
    } else {
      DEBUG(print("symbols not found in libcorkscrew.so\n"));
    }
    dlclose(libcorkscrew);
  } else {
    DEBUG(print("libcorkscrew.so could not be loaded\n"));
  }
  return 0;
}

static int coffeecatch_backtrace_symbols(const backtrace_frame_t* backtrace,
                                         size_t frames,
                                         void (*fun)(void *arg,
                                         const backtrace_symbol_t *sym),
                                         void *arg) {
  int success = 0;
  void *const libcorkscrew = dlopen("libcorkscrew.so", RTLD_LAZY | RTLD_LOCAL);
  if (libcorkscrew != NULL) {
    t_get_backtrace_symbols get_backtrace_symbols 
      = (t_get_backtrace_symbols)
      dlsym(libcorkscrew, "get_backtrace_symbols");
    t_free_backtrace_symbols free_backtrace_symbols 
      = (t_free_backtrace_symbols)
      dlsym(libcorkscrew, "free_backtrace_symbols");
    if (get_backtrace_symbols != NULL
        && free_backtrace_symbols != NULL) {
      backtrace_symbol_t symbols[BACKTRACE_FRAMES_MAX];
      size_t i;
      if (frames > BACKTRACE_FRAMES_MAX) {
        frames = BACKTRACE_FRAMES_MAX;
      }
      get_backtrace_symbols(backtrace, frames, symbols);
      for(i = 0; i < frames; i++) {
        fun(arg, &symbols[i]);
      }
      free_backtrace_symbols(symbols, frames);
      success = 1;
    } else {
      DEBUG(print("symbols not found in libcorkscrew.so\n"));
    }
    dlclose(libcorkscrew);
  } else {
    DEBUG(print("libcorkscrew.so could not be loaded\n"));
  }
  return success;
}
#endif

/* Use libunwind to get a backtrace inside a signal handler.
   Will only return a non-zero code on Android >= 5 (with libunwind.so
   being shipped) */
#ifdef USE_LIBUNWIND
static ssize_t coffeecatch_unwind_signal(siginfo_t* si, void* sc, 
                                         void** frames,
                                         size_t ignore_depth,
                                         size_t max_depth) {
  void *libunwind = dlopen("libunwind.so", RTLD_LAZY | RTLD_LOCAL);
  if (libunwind != NULL) {
    int (*backtrace)(void **buffer, int size) =
      dlsym(libunwind, "unw_backtrace");
    if (backtrace != NULL) {
      int nb = backtrace(frames, max_depth);
      if (nb > 0) {
      }
      return nb;
    } else {
      DEBUG(print("symbols not found in libunwind.so\n"));
    }
    dlclose(libunwind);
  } else {
    DEBUG(print("libunwind.so could not be loaded\n"));
  }
  return -1;
}
#endif

/* Call the old handler. */
static void coffeecatch_call_old_signal_handler(const int code, siginfo_t *const si,
                                                       void * const sc) {
  /* Call the "real" Java handler for JIT and internals. */
  if (code >= 0 && code < SIG_NUMBER_MAX) {
    if (native_code_g.sa_old[code].sa_sigaction != NULL) {
      native_code_g.sa_old[code].sa_sigaction(code, si, sc);
    } else if (native_code_g.sa_old[code].sa_handler != NULL) {
      native_code_g.sa_old[code].sa_handler(code);
    }
  }
}

/* Unflag "on stack" */
static void coffeecatch_revert_alternate_stack(void) {
#ifndef NO_USE_SIGALTSTACK
  stack_t ss;
  if (sigaltstack(NULL, &ss) == 0) {
    ss.ss_flags &= ~SS_ONSTACK;
    sigaltstack (&ss, NULL);
  }
#endif
}

/* Try to jump to userland. */
static void coffeecatch_try_jump_userland(native_code_handler_struct*
                                                 const t,
                                                 const int code,
                                                 siginfo_t *const si,
                                                 void * const sc) {
  (void) si; /* UNUSED */
  (void) sc; /* UNUSED */

  /* Valid context ? */
  if (t != NULL && t->ctx_is_set) {
    DEBUG(print("calling siglongjmp()\n"));

    /* Invalidate the context */
    t->ctx_is_set = 0;

    /* We need to revert the alternate stack before jumping. */
    coffeecatch_revert_alternate_stack();

    /*
     * Note on async-signal-safety of siglongjmp() [POSIX] :
     * "Note that longjmp() and siglongjmp() are not in the list of
     * async-signal-safe functions. This is because the code executing after
     * longjmp() and siglongjmp() can call any unsafe functions with the same
     * danger as calling those unsafe functions directly from the signal
     * handler. Applications that use longjmp() and siglongjmp() from within
     * signal handlers require rigorous protection in order to be portable.
     * Many of the other functions that are excluded from the list are
     * traditionally implemented using either malloc() or free() functions or
     * the standard I/O library, both of which traditionally use data
     * structures in a non-async-signal-safe manner. Since any combination of
     * different functions using a common data structure can cause
     * async-signal-safety problems, this volume of POSIX.1-2008 does not
     * define the behavior when any unsafe function is called in a signal
     * handler that interrupts an unsafe function."
     */
    siglongjmp(t->ctx, code);
  }
}

static void coffeecatch_start_alarm(void) {
  /* Ensure we do not deadlock. Default of ALRM is to die.
   * (signal() and alarm() are signal-safe) */
  (void) alarm(30);
}

static void coffeecatch_mark_alarm(native_code_handler_struct *const t) {
  t->alarm = 1;
}

/* Copy context infos (signal code, etc.) */
static void coffeecatch_copy_context(native_code_handler_struct *const t,
                                     const int code, siginfo_t *const si,
                                     void *const sc) {
  t->code = code;
  t->si = *si;
  if (sc != NULL) {
    ucontext_t *const uc = (ucontext_t*) sc;
    t->uc = *uc;
  } else {
    memset(&t->uc, 0, sizeof(t->uc));
  }

#ifdef USE_UNWIND
  /* Frame buffer initial position. */
  t->frames_size = 0;

  /* Skip us and the caller. */
  t->frames_skip = 2;

  /* Use the corkscrew library to extract the backtrace. */
#ifdef USE_CORKSCREW
  t->frames_size = coffeecatch_backtrace_signal(si, sc, t->frames, 0,
                                                BACKTRACE_FRAMES_MAX);
#else
  /* Unwind frames (equivalent to backtrace()) */
  _Unwind_Backtrace(coffeecatch_unwind_callback, t);
#endif

#ifdef USE_LIBUNWIND
  if (t->frames_size == 0) {
    size_t i;
    t->frames_size = coffeecatch_unwind_signal(si, sc, t->uframes, 0,
                                               BACKTRACE_FRAMES_MAX);
    for(i = 0 ; i < t->frames_size ; i++) {
      t->frames[i].absolute_pc = (uintptr_t) t->uframes[i];
      t->frames[i].stack_top = 0;
      t->frames[i].stack_size = 0;
    }
  }
#endif

  if (t->frames_size != 0) {
    DEBUG(print("called _Unwind_Backtrace()\n"));
  } else {
    DEBUG(print("called _Unwind_Backtrace(), but no traces\n"));
  }
#endif
}

/* Return the thread-specific native_code_handler_struct structure, or
 * @c null if no such structure is available. */
static native_code_handler_struct* coffeecatch_get() {
  return (native_code_handler_struct*)
      pthread_getspecific(native_code_thread);
}

int coffeecatch_cancel_pending_alarm() {
  native_code_handler_struct *const t = coffeecatch_get();
  if (t != NULL && t->alarm) {
    t->alarm = 0;
    /* "If seconds is 0, a pending alarm request, if any, is canceled." */
    alarm(0);
    return 0;
  }
  return -1;
}

/* Internal signal pass-through. Allows to peek the "real" crash before
 * calling the Java handler. Remember than Java needs many of the signals
 * (for the JIT, for test-free NullPointerException handling, etc.)
 * We record the siginfo_t context in this function each time it is being
 * called, to be able to know what error caused an issue.
 */
static void coffeecatch_signal_pass(const int code, siginfo_t *const si,
                                    void *const sc) {
  native_code_handler_struct *t;

  DEBUG(print("caught signal\n"));

  /* Call the "real" Java handler for JIT and internals. */
  coffeecatch_call_old_signal_handler(code, si, sc);

  /* Still here ?
   * FIXME TODO: This is the Dalvik behavior - but is it the SunJVM one ? */

  /* Ensure we do not deadlock. Default of ALRM is to die.
   * (signal() and alarm() are signal-safe) */
  signal(code, SIG_DFL);
  coffeecatch_start_alarm();

  /* Available context ? */
  t = coffeecatch_get();
  if (t != NULL) {
    /* An alarm() call was triggered. */
    coffeecatch_mark_alarm(t);

    /* Take note of the signal. */
    coffeecatch_copy_context(t, code, si, sc);

    /* Back to the future. */
    coffeecatch_try_jump_userland(t, code, si, sc);
  }

  /* Nope. (abort() is signal-safe) */
  DEBUG(print("calling abort()\n"));
  signal(SIGABRT, SIG_DFL);
  abort();
}

/* Internal crash handler for abort(). Java calls abort() if its signal handler
 * could not resolve the signal ; thus calling us through this handler. */
static void coffeecatch_signal_abort(const int code, siginfo_t *const si,
                                            void *const sc) {
  native_code_handler_struct *t;

  (void) sc; /* UNUSED */

  DEBUG(print("caught abort\n"));

  /* Ensure we do not deadlock. Default of ALRM is to die.
   * (signal() and alarm() are signal-safe) */
  signal(code, SIG_DFL);
  coffeecatch_start_alarm();

  /* Available context ? */
  t = coffeecatch_get();
  if (t != NULL) {
    /* An alarm() call was triggered. */
    coffeecatch_mark_alarm(t);

    /* Take note (real "abort()") */
    coffeecatch_copy_context(t, code, si, sc);

    /* Back to the future. */
    coffeecatch_try_jump_userland(t, code, si, sc);
  }

  /* No such restore point, call old signal handler then. */
  DEBUG(print("calling old signal handler\n"));
  coffeecatch_call_old_signal_handler(code, si, sc);

  /* Nope. (abort() is signal-safe) */
  DEBUG(print("calling abort()\n"));
  abort();
}

/* Internal globals initialization. */
static int coffeecatch_handler_setup_global(void) {
  if (native_code_g.initialized++ == 0) {
    size_t i;
    struct sigaction sa_abort;
    struct sigaction sa_pass;

    DEBUG(print("installing global signal handlers\n"));

    /* Setup handler structure. */
    memset(&sa_abort, 0, sizeof(sa_abort));
    sigemptyset(&sa_abort.sa_mask);
    sa_abort.sa_sigaction = coffeecatch_signal_abort;
    sa_abort.sa_flags = SA_SIGINFO | SA_ONSTACK;

    memset(&sa_pass, 0, sizeof(sa_pass));
    sigemptyset(&sa_pass.sa_mask);
    sa_pass.sa_sigaction = coffeecatch_signal_pass;
    sa_pass.sa_flags = SA_SIGINFO | SA_ONSTACK;

    /* Allocate */
    native_code_g.sa_old = calloc(sizeof(struct sigaction), SIG_NUMBER_MAX);
    if (native_code_g.sa_old == NULL) {
      return -1;
    }

    /* Setup signal handlers for SIGABRT (Java calls abort()) and others. **/
    for (i = 0; native_sig_catch[i] != 0; i++) {
      const int sig = native_sig_catch[i];
      const struct sigaction * const action =
          sig == SIGABRT ? &sa_abort : &sa_pass;
      assert(sig < SIG_NUMBER_MAX);
      if (sigaction(sig, action, &native_code_g.sa_old[sig]) != 0) {
        return -1;
      }
    }

    /* Initialize thread var. */
    if (pthread_key_create(&native_code_thread, NULL) != 0) {
      return -1;
    }

    DEBUG(print("installed global signal handlers\n"));
  }

  /* OK. */
  return 0;
}

/**
 * Free a native_code_handler_struct structure.
 **/
static int coffeecatch_native_code_handler_struct_free(native_code_handler_struct *const t) {
  int code = 0;

  if (t == NULL) {
    return -1;
  }

#ifndef NO_USE_SIGALTSTACK
  /* Restore previous alternative stack. */
  if (t->stack_old.ss_sp != NULL && sigaltstack(&t->stack_old, NULL) != 0) {
#ifndef USE_SILENT_SIGALTSTACK
    code = -1;
#endif
  }
#endif

  /* Free alternative stack */
  if (t->stack_buffer != NULL) {
    free(t->stack_buffer);
    t->stack_buffer = NULL;
    t->stack_buffer_size = 0;
  }

  /* Free structure. */
  free(t);

  return code;
}

/**
 * Create a native_code_handler_struct structure.
 **/
static native_code_handler_struct* coffeecatch_native_code_handler_struct_init(void) {
  stack_t stack;
  native_code_handler_struct *const t =
    calloc(sizeof(native_code_handler_struct), 1);

  if (t == NULL) {
    return NULL;
  }

  DEBUG(print("installing thread alternative stack\n"));

  /* Initialize structure */
  t->stack_buffer_size = SIG_STACK_BUFFER_SIZE;
  t->stack_buffer = malloc(t->stack_buffer_size);
  if (t->stack_buffer == NULL) {
    coffeecatch_native_code_handler_struct_free(t);
    return NULL;
  }

  /* Setup alternative stack. */
  memset(&stack, 0, sizeof(stack));
  stack.ss_sp = t->stack_buffer;
  stack.ss_size = t->stack_buffer_size;
  stack.ss_flags = 0;

#ifndef NO_USE_SIGALTSTACK
  /* Install alternative stack. This is thread-safe */
  if (sigaltstack(&stack, &t->stack_old) != 0) {
#ifndef USE_SILENT_SIGALTSTACK
    coffeecatch_native_code_handler_struct_free(t);
    return NULL;
#endif
  }
#endif

  return t;
}

/**
 * Acquire the crash handler for the current thread.
 * The coffeecatch_handler_cleanup() must be called to release allocated
 * resources.
 **/
static int coffeecatch_handler_setup(int setup_thread) {
  int code;

  DEBUG(print("setup for a new handler\n"));

  /* Initialize globals. */
  if (pthread_mutex_lock(&native_code_g.mutex) != 0) {
    return -1;
  }
  code = coffeecatch_handler_setup_global();
  if (pthread_mutex_unlock(&native_code_g.mutex) != 0) {
    return -1;
  }

  /* Global initialization failed. */
  if (code != 0) {
    return -1;
  }

  /* Initialize locals. */
  if (setup_thread && coffeecatch_get() == NULL) {
    native_code_handler_struct *const t =
      coffeecatch_native_code_handler_struct_init();

    if (t == NULL) {
      return -1;
    }

    DEBUG(print("installing thread alternative stack\n"));

    /* Set thread-specific value. */
    if (pthread_setspecific(native_code_thread, t) != 0) {
      coffeecatch_native_code_handler_struct_free(t);
      return -1;
    }

    DEBUG(print("installed thread alternative stack\n"));
  }

  /* OK. */
  return 0;
}

/**
 * Release the resources allocated by a previous call to
 * coffeecatch_handler_setup().
 * This function must be called as many times as
 * coffeecatch_handler_setup() was called to fully release allocated
 * resources.
 **/
static int coffeecatch_handler_cleanup() {
  /* Cleanup locals. */
  native_code_handler_struct *const t = coffeecatch_get();
  if (t != NULL) {
    DEBUG(print("removing thread alternative stack\n"));

    /* Erase thread-specific value now (detach). */
    if (pthread_setspecific(native_code_thread, NULL) != 0) {
      assert(! "pthread_setspecific() failed");
    }

    /* Free handler and reset slternate stack */
    if (coffeecatch_native_code_handler_struct_free(t) != 0) {
      return -1;
    }

    DEBUG(print("removed thread alternative stack\n"));
  }

  /* Cleanup globals. */
  if (pthread_mutex_lock(&native_code_g.mutex) != 0) {
    assert(! "pthread_mutex_lock() failed");
  }
  assert(native_code_g.initialized != 0);
  if (--native_code_g.initialized == 0) {
    size_t i;

    DEBUG(print("removing global signal handlers\n"));

    /* Restore signal handler. */
    for(i = 0; native_sig_catch[i] != 0; i++) {
      const int sig = native_sig_catch[i];
      assert(sig < SIG_NUMBER_MAX);
      if (sigaction(sig, &native_code_g.sa_old[sig], NULL) != 0) {
        return -1;
      }
    }

    /* Free old structure. */
    free(native_code_g.sa_old);
    native_code_g.sa_old = NULL;

    /* Delete thread var. */
    if (pthread_key_delete(native_code_thread) != 0) {
      assert(! "pthread_key_delete() failed");
    }

    DEBUG(print("removed global signal handlers\n"));
  }
  if (pthread_mutex_unlock(&native_code_g.mutex) != 0) {
    assert(! "pthread_mutex_unlock() failed");
  }

  return 0;
}

/**
 * Get the signal associated with the crash.
 */
int coffeecatch_get_signal() {
  const native_code_handler_struct* const t = coffeecatch_get();
  if (t != NULL) {
    return t->code;
  } else {
    return -1;
  }
}

/* Signal descriptions.
   See <http://pubs.opengroup.org/onlinepubs/009696699/basedefs/signal.h.html>
*/
static const char* coffeecatch_desc_sig(int sig, int code) {
  switch(sig) {
  case SIGILL:
    switch(code) {
    case ILL_ILLOPC:
      return "Illegal opcode";
    case ILL_ILLOPN:
      return "Illegal operand";
    case ILL_ILLADR:
      return "Illegal addressing mode";
    case ILL_ILLTRP:
      return "Illegal trap";
    case ILL_PRVOPC:
      return "Privileged opcode";
    case ILL_PRVREG:
      return "Privileged register";
    case ILL_COPROC:
      return "Coprocessor error";
    case ILL_BADSTK:
      return "Internal stack error";
    default:
      return "Illegal operation";
    }
    break;
  case SIGFPE:
    switch(code) {
    case FPE_INTDIV:
      return "Integer divide by zero";
    case FPE_INTOVF:
      return "Integer overflow";
    case FPE_FLTDIV:
      return "Floating-point divide by zero";
    case FPE_FLTOVF:
      return "Floating-point overflow";
    case FPE_FLTUND:
      return "Floating-point underflow";
    case FPE_FLTRES:
      return "Floating-point inexact result";
    case FPE_FLTINV:
      return "Invalid floating-point operation";
    case FPE_FLTSUB:
      return "Subscript out of range";
    default:
      return "Floating-point";
    }
    break;
  case SIGSEGV:
    switch(code) {
    case SEGV_MAPERR:
      return "Address not mapped to object";
    case SEGV_ACCERR:
      return "Invalid permissions for mapped object";
    default:
      return "Segmentation violation";
    }
    break;
  case SIGBUS:
    switch(code) {
    case BUS_ADRALN:
      return "Invalid address alignment";
    case BUS_ADRERR:
      return "Nonexistent physical address";
    case BUS_OBJERR:
      return "Object-specific hardware error";
    default:
      return "Bus error";
    }
    break;
  case SIGTRAP:
    switch(code) {
    case TRAP_BRKPT:
      return "Process breakpoint";
    case TRAP_TRACE:
      return "Process trace trap";
    default:
      return "Trap";
    }
    break;
  case SIGCHLD:
    switch(code) {
    case CLD_EXITED:
      return "Child has exited";
    case CLD_KILLED:
      return "Child has terminated abnormally and did not create a core file";
    case CLD_DUMPED:
      return "Child has terminated abnormally and created a core file";
    case CLD_TRAPPED:
      return "Traced child has trapped";
    case CLD_STOPPED:
      return "Child has stopped";
    case CLD_CONTINUED:
      return "Stopped child has continued";
    default:
      return "Child";
    }
    break;
  case SIGPOLL:
    switch(code) {
    case POLL_IN:
      return "Data input available";
    case POLL_OUT:
      return "Output buffers available";
    case POLL_MSG:
      return "Input message available";
    case POLL_ERR:
      return "I/O error";
    case POLL_PRI:
      return "High priority input available";
    case POLL_HUP:
      return "Device disconnected";
    default:
      return "Pool";
    }
    break;
  case SIGABRT:
    return "Process abort signal";
  case SIGALRM:
    return "Alarm clock";
  case SIGCONT:
    return "Continue executing, if stopped";
  case SIGHUP:
    return "Hangup";
  case SIGINT:
    return "Terminal interrupt signal";
  case SIGKILL:
    return "Kill";
  case SIGPIPE:
    return "Write on a pipe with no one to read it";
  case SIGQUIT:
    return "Terminal quit signal";
  case SIGSTOP:
    return "Stop executing";
  case SIGTERM:
    return "Termination signal";
  case SIGTSTP:
    return "Terminal stop signal";
  case SIGTTIN:
    return "Background process attempting read";
  case SIGTTOU:
    return "Background process attempting write";
  case SIGUSR1:
    return "User-defined signal 1";
  case SIGUSR2:
    return "User-defined signal 2";
  case SIGPROF:
    return "Profiling timer expired";
  case SIGSYS:
    return "Bad system call";
  case SIGVTALRM:
    return "Virtual timer expired";
  case SIGURG:
    return "High bandwidth data is available at a socket";
  case SIGXCPU:
    return "CPU time limit exceeded";
  case SIGXFSZ:
    return "File size limit exceeded";
  default:
    switch(code) {
    case SI_USER:
      return "Signal sent by kill()";
    case SI_QUEUE:
      return "Signal sent by the sigqueue()";
    case SI_TIMER:
      return "Signal generated by expiration of a timer set by timer_settime()";
    case SI_ASYNCIO:
      return "Signal generated by completion of an asynchronous I/O request";
    case SI_MESGQ:
      return
        "Signal generated by arrival of a message on an empty message queue";
    default:
      return "Unknown signal";
    }
    break;
  }
}

/**
 * Get the backtrace size. Returns 0 if no backtrace is available.
 */
size_t coffeecatch_get_backtrace_size(void) {
#ifdef USE_UNWIND
  const native_code_handler_struct* const t = coffeecatch_get();
  if (t != NULL) {
    return t->frames_size;
  } else {
    return 0;
  }
#else
  return 0;
#endif
}

/**
 * Get the <index>th element of the backtrace, or 0 upon error.
 */
uintptr_t coffeecatch_get_backtrace(ssize_t index) {
#ifdef USE_UNWIND
  const native_code_handler_struct* const t = coffeecatch_get();
  if (t != NULL) {
    if (index < 0) {
      index = t->frames_size + index;
    }
    if (index >= 0 && (size_t) index < t->frames_size) {
#ifdef USE_CORKSCREW
      return t->frames[index].absolute_pc;
#else
      return t->frames[index];
#endif
    }
  }
#else
  (void) index;
#endif
  return 0;
}

/**
 * Get the program counter, given a pointer to a ucontext_t context.
 **/
static uintptr_t coffeecatch_get_pc_from_ucontext(const ucontext_t *uc) {
#if (defined(__arm__))
  return uc->uc_mcontext.arm_pc;
#elif defined(__aarch64__)
  return uc->uc_mcontext.pc;
#elif (defined(__x86_64__))
  return uc->uc_mcontext.gregs[REG_RIP];
#elif (defined(__i386))
  return uc->uc_mcontext.gregs[REG_EIP];
#elif (defined (__ppc__)) || (defined (__powerpc__))
  return uc->uc_mcontext.regs->nip;
#elif (defined(__hppa__))
  return uc->uc_mcontext.sc_iaoq[0] & ~0x3UL;
#elif (defined(__sparc__) && defined (__arch64__))
  return uc->uc_mcontext.mc_gregs[MC_PC];
#elif (defined(__sparc__) && !defined (__arch64__))
  return uc->uc_mcontext.gregs[REG_PC];
#elif (defined(__mips__))
  return uc->uc_mcontext.gregs[31];
#else
#error "Architecture is unknown, please report me!"
#endif
}

/* Is this module name look like a DLL ?
   FIXME: find a better way to do that...  */
static int coffeecatch_is_dll(const char *name) {
  size_t i;
  for(i = 0; name[i] != '\0'; i++) {
    if (name[i + 0] == '.' &&
        name[i + 1] == 's' &&
        name[i + 2] == 'o' &&
        ( name[i + 3] == '\0' || name[i + 3] == '.') ) {
      return 1;
    }
  }
  return 0;
}

/* Extract a line information on a PC address. */
static void format_pc_address_cb(uintptr_t pc, 
                                 void (*fun)(void *arg, const char *module, 
                                             uintptr_t addr,
                                             const char *function,
                                             uintptr_t offset), void *arg) {
  if (pc != 0) {
    Dl_info info;
    void * const addr = (void*) pc;
    /* dladdr() returns 0 on error, and nonzero on success. */
    if (dladdr(addr, &info) != 0 && info.dli_fname != NULL) {
      const uintptr_t near = (uintptr_t) info.dli_saddr;
      const uintptr_t offs = pc - near;
      const uintptr_t addr_rel = pc - (uintptr_t) info.dli_fbase;
      /* We need the absolute address for the main module (?).
         TODO FIXME to be investigated. */
      const uintptr_t addr_to_use = coffeecatch_is_dll(info.dli_fname)
        ? addr_rel : pc;
      fun(arg, info.dli_fname, addr_to_use, info.dli_sname, offs);
    } else {
      fun(arg, NULL, pc, NULL, 0);
    }
  }
}

typedef struct t_print_fun {
  char *buffer;
  size_t buffer_size;
} t_print_fun;

static void print_fun(void *arg, const char *module, uintptr_t uaddr,
                      const char *function, uintptr_t offset) {
  t_print_fun *const t = (t_print_fun*) arg;
  char *const buffer = t->buffer;
  const size_t buffer_size = t->buffer_size;
  const void*const addr = (void*) uaddr;
  if (module == NULL) {
    snprintf(buffer, buffer_size, "[at %p]", addr);
  } else if (function != NULL) {
    snprintf(buffer, buffer_size, "[at %s:%p (%s+0x%x)]", module, addr,
             function, (int) offset);
  } else {
    snprintf(buffer, buffer_size, "[at %s:%p]", module, addr);
  }
}

/* Format a line information on a PC address. */
static void format_pc_address(char *buffer, size_t buffer_size, uintptr_t pc) {
  t_print_fun t;
  t.buffer = buffer;
  t.buffer_size = buffer_size;
  format_pc_address_cb(pc, print_fun, &t);
}

/**
 * Get the full error message associated with the crash.
 */
const char* coffeecatch_get_message() {
  const int error = errno;
  const native_code_handler_struct* const t = coffeecatch_get();

  /* Found valid handler. */
  if (t != NULL) {
    char * const buffer = t->stack_buffer;
    const size_t buffer_len = t->stack_buffer_size;
    size_t buffer_offs = 0;

    const char* const posix_desc =
      coffeecatch_desc_sig(t->si.si_signo, t->si.si_code);

    /* Assertion failure ? */
    if ((t->code == SIGABRT
#ifdef __ANDROID__
        /* See Android BUG #16672:
         * "C assert() failure causes SIGSEGV when it should cause SIGABRT" */
        || (t->code == SIGSEGV && (uintptr_t) t->si.si_addr == 0xdeadbaad)
#endif
        ) && t->expression != NULL) {
      snprintf(&buffer[buffer_offs], buffer_len - buffer_offs,
          "assertion '%s' failed at %s:%d",
          t->expression, t->file, t->line);
      buffer_offs += strlen(&buffer[buffer_offs]);
    }
    /* Signal */
    else {
      snprintf(&buffer[buffer_offs], buffer_len - buffer_offs, "signal %d",
               t->si.si_signo);
      buffer_offs += strlen(&buffer[buffer_offs]);

      /* Description */
      snprintf(&buffer[buffer_offs], buffer_len - buffer_offs, " (%s)",
               posix_desc);
      buffer_offs += strlen(&buffer[buffer_offs]);

      /* Address of faulting instruction */
      if (t->si.si_signo == SIGILL || t->si.si_signo == SIGSEGV) {
        snprintf(&buffer[buffer_offs], buffer_len - buffer_offs, " at address %p",
                 t->si.si_addr);
        buffer_offs += strlen(&buffer[buffer_offs]);
      }
    }

    /* [POSIX] If non-zero, an errno value associated with this signal,
     as defined in <errno.h>. */
    if (t->si.si_errno != 0) {
      snprintf(&buffer[buffer_offs], buffer_len - buffer_offs, ": ");
      buffer_offs += strlen(&buffer[buffer_offs]);
      if (strerror_r(t->si.si_errno, &buffer[buffer_offs],
                     buffer_len - buffer_offs) == 0) {
        snprintf(&buffer[buffer_offs], buffer_len - buffer_offs,
                 "unknown error");
        buffer_offs += strlen(&buffer[buffer_offs]);
      }
    }

    /* Sending process ID. */
    if (t->si.si_signo == SIGCHLD && t->si.si_pid != 0) {
      snprintf(&buffer[buffer_offs], buffer_len - buffer_offs,
               " (sent by pid %d)", (int) t->si.si_pid);
      buffer_offs += strlen(&buffer[buffer_offs]);
    }

    /* Faulting program counter location. */
    if (coffeecatch_get_pc_from_ucontext(&t->uc) != 0) {
      const uintptr_t pc = coffeecatch_get_pc_from_ucontext(&t->uc);
      snprintf(&buffer[buffer_offs], buffer_len - buffer_offs, " ");
      buffer_offs += strlen(&buffer[buffer_offs]);
      format_pc_address(&buffer[buffer_offs], buffer_len - buffer_offs, pc);
      buffer_offs += strlen(&buffer[buffer_offs]);
    }

    /* Return string. */
    buffer[buffer_offs] = '\0';
    return t->stack_buffer;
  } else {
    /* Static buffer in case of emergency */
    static char buffer[256];
#ifdef _GNU_SOURCE
    return strerror_r(error, &buffer[0], sizeof(buffer));
#else
    const int code = strerror_r(error, &buffer[0], sizeof(buffer));
    errno = error;
    if (code == 0) {
      return buffer;
    } else {
      return "unknown error during crash handler setup";
    }
#endif
  }
}

#if (defined(USE_CORKSCREW))
typedef struct t_coffeecatch_backtrace_symbols_fun {
  void (*fun)(void *arg, const char *module, uintptr_t addr,
              const char *function, uintptr_t offset);
  void *arg;
} t_coffeecatch_backtrace_symbols_fun;

static void coffeecatch_backtrace_symbols_fun(void *arg, const backtrace_symbol_t *sym) {
  t_coffeecatch_backtrace_symbols_fun *const bt =
    (t_coffeecatch_backtrace_symbols_fun*) arg;
  const char *symbol = sym->demangled_name != NULL 
    ? sym->demangled_name : sym->symbol_name;
  const uintptr_t rel = sym->relative_pc - sym->relative_symbol_addr;
  bt->fun(bt->arg, sym->map_name, sym->relative_pc, symbol, rel);
}
#endif

/**
 * Enumerate backtrace information.
 */
void coffeecatch_get_backtrace_info(void (*fun)(void *arg,
                                    const char *module,
                                    uintptr_t addr,
                                    const char *function,
                                    uintptr_t offset), void *arg) {
  const native_code_handler_struct* const t = coffeecatch_get();
  if (t != NULL) {
    size_t i;
#if (defined(USE_CORKSCREW))
    t_coffeecatch_backtrace_symbols_fun bt;
    bt.fun = fun;
    bt.arg = arg;
    if (coffeecatch_backtrace_symbols(t->frames, t->frames_size,
                                      coffeecatch_backtrace_symbols_fun,
                                      &bt)) {
      return;
    }
#endif
    for(i = 0; i < t->frames_size; i++) {
      const uintptr_t pc = t->frames[i].absolute_pc;
      format_pc_address_cb(pc, fun, arg);
    }
  }
}

/**
 * Returns 1 if we are already inside a coffeecatch block, 0 otherwise.
 */
int coffeecatch_inside() {
  native_code_handler_struct *const t = coffeecatch_get();
  if (t != NULL && t->reenter > 0) {
    t->reenter++;
    return 1;
  }
  return 0;
}

/**
 * Calls coffeecatch_handler_setup(1) to setup a crash handler, mark the
 * context as valid, and return 0 upon success.
 */
int coffeecatch_setup() {
  if (coffeecatch_handler_setup(1) == 0) {
    native_code_handler_struct *const t = coffeecatch_get();
    assert(t != NULL);
    assert(t->reenter == 0);
    t->reenter = 1;
    t->ctx_is_set = 1;
    return 0;
  } else {
    return -1;
  }
}

/**
 * Calls coffeecatch_handler_cleanup()
 */
void coffeecatch_cleanup() {
  native_code_handler_struct *const t = coffeecatch_get();
  assert(t != NULL);
  assert(t->reenter > 0);
  t->reenter--;
  if (t->reenter == 0) {
    t->ctx_is_set = 0;
    coffeecatch_handler_cleanup();
  }
}

sigjmp_buf* coffeecatch_get_ctx() {
  native_code_handler_struct* t = coffeecatch_get();
  assert(t != NULL);
  return &t->ctx;
}

void coffeecatch_abort(const char* exp, const char* file, int line) {
  native_code_handler_struct *const t = coffeecatch_get();
  if (t != NULL) {
    t->expression = exp;
    t->file = file;
    t->line = line;
  }
  abort();
}
