#include "textflag.h"

// func spreadingHistNEON(x []int32, n int) (t0, t1, t2 int)
//
// Per position j in [0,n): xj = int16(x[j] >> 10); p = xj*xj (int32);
// sq = int16(p >> 15); x2N = sq * int16(n) (int32); then t0/t1/t2 count
// x2N < 2048 / 512 / 128. The >>10 is NORM_SHIFT-14 (NORM_SHIFT is the frozen
// libopus 24, see vq.go), hardcoded as it must be in hand asm; the generic
// reference uses the symbolic normShift-14 and TestSpreadingDecisionMatchesC
// pins the two together. The vector ops are WORD-encoded (Go's arm64 assembler
// lacks these mnemonics); each WORD is the clang encoding for the instruction in
// its comment. The register numbers are baked into the encodings, so the
// allocation is fixed: R0=x ptr (ld1 [x0]), R2=broadcast source (dup wN),
// R6/R7/R8=umov outputs, V0-V4 pipeline, V5=int16(n), V6/V7/V16=masks,
// V17/V18/V19=accumulators, V20/V21/V22=thresholds. XTN truncates (not SQXTN),
// matching Go's int16(...).
//
// Layout: x_base+0, x_len+8, x_cap+16, n+24, t0+32, t1+40, t2+48.
TEXT ·spreadingHistNEON(SB), NOSPLIT, $0-56
	MOVD	x_base+0(FP), R0	// R0 = &x[0]  (ld1 uses x0)
	MOVD	n+24(FP), R1		// R1 = remaining count

	MOVD	$0, R3			// t0 scalar
	MOVD	$0, R4			// t1 scalar
	MOVD	$0, R5			// t2 scalar

	MOVD	n+24(FP), R9		// R9 = int16(n) for the scalar tail multiply
	SXTH	R9, R9

	WORD	$0x4f000411		// movi v17.4s, #0   (acc0)
	WORD	$0x4f000412		// movi v18.4s, #0   (acc1)
	WORD	$0x4f000413		// movi v19.4s, #0   (acc2)

	MOVD	$2048, R2
	WORD	$0x4e040c54		// dup v20.4s, w2    (thr0)
	MOVD	$512, R2
	WORD	$0x4e040c55		// dup v21.4s, w2    (thr1)
	MOVD	$128, R2
	WORD	$0x4e040c56		// dup v22.4s, w2    (thr2)
	MOVD	n+24(FP), R2
	WORD	$0x0e020c45		// dup v5.4h, w2     (int16(n) x4)

vecloop:
	CMP	$4, R1
	BLT	tail
	WORD	$0x4cdf7800		// ld1 {v0.4s}, [x0], #16
	WORD	$0x4f360400		// sshr v0.4s, v0.4s, #10
	WORD	$0x0e612801		// xtn v1.4h, v0.4s
	WORD	$0x0e61c022		// smull v2.4s, v1.4h, v1.4h
	WORD	$0x4f310442		// sshr v2.4s, v2.4s, #15
	WORD	$0x0e612843		// xtn v3.4h, v2.4s
	WORD	$0x0e65c064		// smull v4.4s, v3.4h, v5.4h
	WORD	$0x4ea43686		// cmgt v6.4s, v20.4s, v4.4s   (2048 > x2N)
	WORD	$0x4ea436a7		// cmgt v7.4s, v21.4s, v4.4s   (512 > x2N)
	WORD	$0x4ea436d0		// cmgt v16.4s, v22.4s, v4.4s  (128 > x2N)
	WORD	$0x6ea68631		// sub v17.4s, v17.4s, v6.4s   (acc0 += count)
	WORD	$0x6ea78652		// sub v18.4s, v18.4s, v7.4s
	WORD	$0x6eb08673		// sub v19.4s, v19.4s, v16.4s
	SUB	$4, R1
	B	vecloop

tail:
	CBZ	R1, reduce
scalarloop:
	MOVW	(R0), R10		// x[j] (sign-extended int32)
	ASR	$10, R10, R10		// x[j] >> 10
	SXTH	R10, R10		// xj = int16(x[j]>>10)
	MUL	R10, R10, R10		// xj*xj
	ASR	$15, R10, R10		// >> 15
	SXTH	R10, R10		// sq = int16((xj*xj)>>15)
	MUL	R9, R10, R10		// x2N = sq * int16(n)
	CMP	$2048, R10
	BGE	c1
	ADD	$1, R3, R3
c1:
	CMP	$512, R10
	BGE	c2
	ADD	$1, R4, R4
c2:
	CMP	$128, R10
	BGE	c3
	ADD	$1, R5, R5
c3:
	ADD	$4, R0
	SUB	$1, R1
	CBNZ	R1, scalarloop

reduce:
	WORD	$0x4eb1ba31		// addv s17, v17.4s
	WORD	$0x4eb1ba52		// addv s18, v18.4s
	WORD	$0x4eb1ba73		// addv s19, v19.4s
	WORD	$0x0e043e26		// mov w6, v17.s[0]
	WORD	$0x0e043e47		// mov w7, v18.s[0]
	WORD	$0x0e043e68		// mov w8, v19.s[0]
	ADD	R6, R3, R3
	ADD	R7, R4, R4
	ADD	R8, R5, R5
	MOVD	R3, t0+32(FP)
	MOVD	R4, t1+40(FP)
	MOVD	R5, t2+48(FP)
	RET
