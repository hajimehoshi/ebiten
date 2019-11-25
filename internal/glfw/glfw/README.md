These files are basically copy of github.com/v3.3/glfw/glfw.

There is one change from the original files: `GLFWscrollfun` takes pointers instead of values since all arguments of C functions have to be 32bit on 32bit Windows machine.
