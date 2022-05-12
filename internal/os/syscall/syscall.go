package syscall

import "unsafe"

func SyscallX(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	return syscall_syscallX(fn, a1, a2, a3)
}

var syscallX6FABI0 uintptr

func syscallX6F() // implemented in assembly

func SyscallXF(fn, a1, a2, a3 uintptr, f1, f2, f3 float64) (r1, r2, err uintptr) {
	args := struct {
		fn, a1, a2, a3 uintptr
		f1, f2, f3     float64
		r1, r2, err    uintptr
	}{fn, a1, a2, a3, f1, f2, f3, r1, r2, err}
	runtime_entersyscall()
	runtime_libcCall(unsafe.Pointer(syscallX6FABI0), unsafe.Pointer(&args))
	runtime_exitsyscall()
	return args.r1, args.r2, args.err
}

var syscallX6ABI0 uintptr

func syscallX6() // implemented in assembly

func SyscallX6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr) {
	args := struct{ fn, a1, a2, a3, a4, a5, a6, r1, r2, err uintptr }{fn, a1, a2, a3, a4, a5, a6, r1, r2, err}
	runtime_entersyscall()
	runtime_libcCall(unsafe.Pointer(syscallX6ABI0), unsafe.Pointer(&args))
	runtime_exitsyscall()
	return args.r1, args.r2, args.err
}

var syscallX9ABI0 uintptr

func syscallX9() // implemented in assembly

func SyscallX9(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2, err uintptr) {
	args := struct{ fn, a1, a2, a3, a4, a5, a6, a7, a8, a9, r1, r2, err uintptr }{
		fn, a1, a2, a3, a4, a5, a6, a7, a8, a9, r1, r2, err}
	runtime_entersyscall()
	runtime_libcCall(unsafe.Pointer(syscallX9ABI0), unsafe.Pointer(&args))
	runtime_exitsyscall()
	return args.r1, args.r2, args.err
}
