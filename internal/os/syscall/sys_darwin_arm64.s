#include "textflag.h"

// syscallX6 calls a function in libc on behalf of the syscall package.
// syscallX6 takes a pointer to a struct like:
// struct {
//	fn    uintptr
//	a1    uintptr
//	a2    uintptr
//	a3    uintptr
//	a4    uintptr
//	a5    uintptr
//	a6    uintptr
//	r1    uintptr
//	r2    uintptr
//	err   uintptr
// }
// syscallX6 must be called on the g0 stack with the
// C calling convention (use libcCall).
GLOBL ·syscallX6ABI0(SB), NOPTR|RODATA, $8
DATA ·syscallX6ABI0(SB)/8, $·syscallX6(SB)
TEXT ·syscallX6(SB),NOSPLIT,$0
	SUB	$16, RSP	// push structure pointer
	MOVD	R0, (RSP)

	MOVD	0(R0), R12	// fn
	MOVD	16(R0), R1	// a2
	MOVD	24(R0), R2	// a3
	MOVD	32(R0), R3	// a4
	MOVD	40(R0), R4	// a5
	MOVD	48(R0), R5	// a6
	MOVD	8(R0), R0	// a1
	BL	(R12)

	MOVD	(RSP), R2	// pop structure pointer
	ADD	$16, RSP
	MOVD	R0, 56(R2)	// save r1
	MOVD	R1, 64(R2)	// save r2
	CMP	$-1, R0
	BNE	ok
	SUB	$16, RSP	// push structure pointer
	MOVD	R2, (RSP)
	BL	libc_error(SB)
	MOVW	(R0), R0
	MOVD	(RSP), R2	// pop structure pointer
	ADD	$16, RSP
	MOVD	R0, 72(R2)	// save err
ok:
	RET

// syscallXF calls a function in libc on behalf of the syscall package.
// syscallXF takes a pointer to a struct like:
// struct {
//	fn    uintptr
//	a1    uintptr
//	a2    uintptr
//	a3    uintptr
//	f1    float64
//	f2    float64
//	f3    float64
//	r1    uintptr
//	r2    uintptr
//	err   uintptr
// }
// syscallXF must be called on the g0 stack with the
// C calling convention (use libcCall).
GLOBL ·syscallXFABI0(SB), NOPTR|RODATA, $8
DATA ·syscallXFABI0(SB)/8, $·syscallXF(SB)
TEXT ·syscallXF(SB),NOSPLIT,$0
	SUB	$16, RSP	// push structure pointer
	MOVD	R0, (RSP)

	MOVD	0(R0), R12	// fn
	MOVD	16(R0), R1	// a2
	MOVD	24(R0), R2	// a3
	FMOVD	32(R0), F0	// f1
	FMOVD	40(R0), F1	// f2
	FMOVD	48(R0), F2	// f3
	MOVD	8(R0), R0	// a1
	BL	(R12)

	MOVD	(RSP), R2	// pop structure pointer
	ADD	$16, RSP
	MOVD	R0, 56(R2)	// save r1
	MOVD	R1, 64(R2)	// save r2
	CMP	$-1, R0
	BNE	ok
	SUB	$16, RSP	// push structure pointer
	MOVD	R2, (RSP)
	BL	libc_error(SB)
	MOVW	(R0), R0
	MOVD	(RSP), R2	// pop structure pointer
	ADD	$16, RSP
	MOVD	R0, 72(R2)	// save err
ok:
	RET

// syscallX9 calls a function in libc on behalf of the syscall package.
// syscallX9 takes a pointer to a struct like:
// struct {
//	fn    uintptr
//	a1    uintptr
//	a2    uintptr
//	a3    uintptr
//	a4    uintptr
//	a5    uintptr
//	a6    uintptr
//	a7    uintptr
//	a8    uintptr
//	a9    uintptr
//	r1    uintptr
//	r2    uintptr
//	err   uintptr
// }
// syscallX9 must be called on the g0 stack with the
// C calling convention (use libcCall).
GLOBL ·syscallX9ABI0(SB), NOPTR|RODATA, $8
DATA ·syscallX9ABI0(SB)/8, $·syscallX9(SB)
TEXT ·syscallX9(SB),NOSPLIT,$0
	SUB	$16, RSP	// push structure pointer
	MOVD	R0, 8(RSP)

	MOVD	0(R0), R12	// fn
	MOVD	16(R0), R1	// a2
	MOVD	24(R0), R2	// a3
	MOVD	32(R0), R3	// a4
	MOVD	40(R0), R4	// a5
	MOVD	48(R0), R5	// a6
	MOVD	56(R0), R6	// a7
	MOVD	64(R0), R7	// a8
	MOVD	72(R0), R8	// a9
	MOVD    R8, (RSP)   // push a9 onto stack
	MOVD	8(R0), R0	// a1
	BL	(R12)

	MOVD	8(RSP), R2	// pop structure pointer
	ADD	$16, RSP
	MOVD	R0, 80(R2)	// save r1
	MOVD	R1, 88(R2)	// save r2
	CMP	$-1, R0
	BNE	ok
	SUB	$16, RSP	// push structure pointer
	MOVD	R2, (RSP)
	BL	libc_error(SB)
	MOVW	(R0), R0
	MOVD	(RSP), R2	// pop structure pointer
	ADD	$16, RSP
	MOVD	R0, 96(R2)	// save err
ok:
	RET
