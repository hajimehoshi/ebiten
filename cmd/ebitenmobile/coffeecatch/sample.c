#include <stdio.h>
#include <stdlib.h>

#include "coffeecatch.h"

int recurse_madness(int level) {
  static int var[] = { 1, 2 };
  if (level > 2000) {
    return 1 + level;
  } else {
    return recurse_madness(level + 1)*var[level];
  }
}

static char string_buffer[256];

static __attribute__ ((noinline)) void demo(int *fault) {
  COFFEE_TRY() {
    recurse_madness(42);
    *fault = 0;
  } COFFEE_CATCH() {
    const char*const message = coffeecatch_get_message();
    snprintf(string_buffer, sizeof(string_buffer), "%s", message);
    *fault = 1;
  } COFFEE_END();
}

int main(int argc, char **argv) {
  int fault;
  (void) argc;
  (void) argv;

  printf("running demo...\n");
  demo(&fault);
  if (fault != 0) {
    fprintf(stderr, "** crash detected: %s\n", string_buffer);
    exit(EXIT_FAILURE);
  }
  printf("success!\n");

  return EXIT_SUCCESS;
}

