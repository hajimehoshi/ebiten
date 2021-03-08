These files are basically copy of github.com/v3.3/glfw/glfw.

There is one change from the original files: `GLFWscrollfun` takes pointers instead of values since all arguments of C functions have to be 32bit on 32bit Windows machine.

```diff
diff --git a/tmp/glfw-3.3.3/include/GLFW/glfw3.h b/./internal/glfw/glfw/include/GLFW/glfw3.h
index 35bbf075..b41c0dca 100644
--- a/tmp/glfw-3.3.3/include/GLFW/glfw3.h
+++ b/./internal/glfw/glfw/include/GLFW/glfw3.h
@@ -1496,7 +1496,7 @@ typedef void (* GLFWcursorenterfun)(GLFWwindow*,int);
  *
  *  @ingroup input
  */
-typedef void (* GLFWscrollfun)(GLFWwindow*,double,double);
+typedef void (* GLFWscrollfun)(GLFWwindow*,double*,double*);

 /*! @brief The function pointer type for keyboard key callbacks.
  *
```

```diff
diff --git a/tmp/glfw-3.3.3/src/input.c b/./internal/glfw/glfw/src/input.c
index 337d5cf0..4ac555cb 100644
--- a/tmp/glfw-3.3.3/src/input.c
+++ b/./internal/glfw/glfw/src/input.c
@@ -312,7 +312,7 @@ void _glfwInputChar(_GLFWwindow* window, unsigned int codepoint, int mods, GLFWb
 void _glfwInputScroll(_GLFWwindow* window, double xoffset, double yoffset)
 {
     if (window->callbacks.scroll)
-        window->callbacks.scroll((GLFWwindow*) window, xoffset, yoffset);
+        window->callbacks.scroll((GLFWwindow*) window, &xoffset, &yoffset);
 }

 // Notifies shared code of a mouse button click event
```
