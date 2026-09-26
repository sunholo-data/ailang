#include "textflag.h"

// func spreadingHistSSE2(x []int32, n int) (t0, t1, t2 int)
//
// Per position j in [0,n): xj = int16(x[j] >> 10); p = xj*xj (int32);
// sq = int16(p >> 15); x2N = sq * int16(n) (int32); then t0/t1/t2 count
// x2N < 2048 / 512 / 128. The >>10 is NORM_SHIFT-14 (NORM_SHIFT is the frozen
// libopus 24, see vq.go), hardcoded as it must be in hand asm; the generic
// reference uses the symbolic normShift-14 and TestSpreadingDecisionMatchesC
// pins the two together. Both int16 truncations are done with an arithmetic
// shift, a low-16 mask, and PMADDWL (signed int16*int16 -> int32), which wraps
// exactly as Go's int16(...) does; a saturating PACKSSDW would NOT match.
//
// Layout: x_base+0, x_len+8, x_cap+16, n+24, t0+32, t1+40, t2+48.
TEXT ·spreadingHistSSE2(SB), NOSPLIT, $0-56
	MOVQ	x_base+0(FP), AX	// AX = &x[0]
	MOVQ	n+24(FP), BX		// BX = remaining count

	XORQ	R8, R8			// t0 scalar
	XORQ	R9, R9			// t1 scalar
	XORQ	R10, R10		// t2 scalar

	PXOR	X5, X5			// acc0 vector
	PXOR	X6, X6			// acc1 vector
	PXOR	X7, X7			// acc2 vector

	MOVQ	n+24(FP), SI		// SI = int16(n) sign-extended (scalar multiply)
	MOVWLSX	SI, SI

	MOVL	$2048, DX
	MOVD	DX, X8
	PSHUFL	$0, X8, X8		// X8 = [2048]x4
	MOVL	$512, DX
	MOVD	DX, X9
	PSHUFL	$0, X9, X9		// X9 = [512]x4
	MOVL	$128, DX
	MOVD	DX, X10
	PSHUFL	$0, X10, X10		// X10 = [128]x4
	MOVL	$0x0000FFFF, DX
	MOVD	DX, X11
	PSHUFL	$0, X11, X11		// X11 = low-16 mask
	MOVQ	n+24(FP), DX
	MOVD	DX, X12
	PSHUFL	$0, X12, X12		// X12 = [n]x4; PMADDWL below reads low16 = int16(n)
					// (high16 is 0 for the n < 65536 bands ever seen, and
					// harmless otherwise since sq's paired high16 is masked to 0)

vecloop:
	CMPQ	BX, $4
	JL	tail
	MOVOU	(AX), X0		// x[j..j+3]
	PSRAL	$10, X0			// >> 10 arithmetic
	PAND	X11, X0			// [low16(xj), 0] per lane
	MOVO	X0, X1
	PMADDWL	X0, X1			// X1 = xj*xj (int32 lanes)
	PSRAL	$15, X1			// >> 15
	PAND	X11, X1			// [low16(sq), 0]
	PMADDWL	X12, X1			// X1 = sq*int16(n) = x2N
	MOVO	X8, X2
	PCMPGTL	X1, X2			// X2 = (2048 > x2N) mask
	PSUBL	X2, X5			// acc0 -= mask (adds count)
	MOVO	X9, X3
	PCMPGTL	X1, X3
	PSUBL	X3, X6
	MOVO	X10, X4
	PCMPGTL	X1, X4
	PSUBL	X4, X7
	ADDQ	$16, AX
	SUBQ	$4, BX
	JMP	vecloop

tail:
	TESTQ	BX, BX
	JZ	reduce
scalarloop:
	MOVL	(AX), DX		// x[j]
	SARL	$10, DX
	MOVWLSX	DX, DX			// xj = int16(x[j]>>10)
	IMULL	DX, DX			// xj*xj
	SARL	$15, DX
	MOVWLSX	DX, DX			// sq = int16((xj*xj)>>15)
	IMULL	SI, DX			// x2N = sq * int16(n)
	CMPL	DX, $2048
	JGE	c1
	INCQ	R8
c1:
	CMPL	DX, $512
	JGE	c2
	INCQ	R9
c2:
	CMPL	DX, $128
	JGE	c3
	INCQ	R10
c3:
	ADDQ	$4, AX
	DECQ	BX
	JNZ	scalarloop

reduce:
	// MOVD xmm->gpr is 64-bit in Go asm, so after the horizontal sum leaves the
	// total in lane 0, MOVL truncates to lane 0 (zeroing the high dword = lane 1).
	MOVO	X5, X0			// horizontal-sum acc0 -> R8
	PSRLO	$8, X0
	PADDL	X0, X5
	MOVO	X5, X0
	PSRLO	$4, X0
	PADDL	X0, X5
	MOVD	X5, DX
	MOVL	DX, DX
	ADDQ	DX, R8
	MOVO	X6, X0			// acc1 -> R9
	PSRLO	$8, X0
	PADDL	X0, X6
	MOVO	X6, X0
	PSRLO	$4, X0
	PADDL	X0, X6
	MOVD	X6, DX
	MOVL	DX, DX
	ADDQ	DX, R9
	MOVO	X7, X0			// acc2 -> R10
	PSRLO	$8, X0
	PADDL	X0, X7
	MOVO	X7, X0
	PSRLO	$4, X0
	PADDL	X0, X7
	MOVD	X7, DX
	MOVL	DX, DX
	ADDQ	DX, R10

	MOVQ	R8, t0+32(FP)
	MOVQ	R9, t1+40(FP)
	MOVQ	R10, t2+48(FP)
	RET
