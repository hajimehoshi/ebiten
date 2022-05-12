package dl

var dlopenABI0 uintptr
var dlsymABI0 uintptr
var dlerrorABI0 uintptr
var dlcloseABI0 uintptr

func dlopen(path *byte, mode int) (ret uintptr)

func dlerror() (ret uintptr)

func dlclose(handle uintptr) (ret int)

func dlsym(handle uintptr, symbol *byte) (ret uintptr)
