package smt

import "fmt"

// GenerateListUnrolling generates SMT-LIB declarations for bounded recursive
// list operations. Returns a slice of SMT-LIB declaration strings, or nil if
// the operation is not recognized.
//
// For depth N, produces N+1 declarations:
//   - Level 0: (declare-fun op_0 ...) — uninterpreted function
//   - Levels 1..N: (define-fun op_k ...) where recursive calls use op_(k-1)
//
// Supported operations: _list_reverse, _list_take, _list_drop
func GenerateListUnrolling(op string, depth int, elemSort string) []string {
	seqSort := fmt.Sprintf("(Seq %s)", elemSort)

	switch op {
	case "_list_reverse":
		return generateReverseUnrolling(depth, elemSort, seqSort)
	case "_list_take":
		return generateTakeUnrolling(depth, elemSort, seqSort)
	case "_list_drop":
		return generateDropUnrolling(depth, elemSort, seqSort)
	default:
		return nil
	}
}

// TopLevelUnrolledName returns the name of the top-level (deepest) unrolled
// function for a given operation and depth.
func TopLevelUnrolledName(op string, depth int) string {
	return fmt.Sprintf("%s_%d", op, depth)
}

// generateReverseUnrolling generates bounded unrolling for _list_reverse.
//
// Level 0: (declare-fun _list_reverse_0 ((Seq Int)) (Seq Int))
// Level k: (define-fun _list_reverse_k ((xs (Seq Int))) (Seq Int)
//
//	(ite (= (seq.len xs) 0)
//	  (as seq.empty (Seq Int))
//	  (seq.++ (_list_reverse_(k-1) (seq.extract xs 1 (- (seq.len xs) 1)))
//	          (seq.unit (seq.nth xs 0)))))
func generateReverseUnrolling(depth int, elemSort, seqSort string) []string {
	decls := make([]string, 0, depth+1)

	// Level 0: uninterpreted
	level0 := fmt.Sprintf("(declare-fun _list_reverse_0 (%s) %s)", seqSort, seqSort)
	decls = append(decls, level0)

	// Levels 1..depth
	for k := 1; k <= depth; k++ {
		name := fmt.Sprintf("_list_reverse_%d", k)
		prev := fmt.Sprintf("_list_reverse_%d", k-1)
		body := fmt.Sprintf(
			"(define-fun %s ((xs %s)) %s\n"+
				"  (ite (= (seq.len xs) 0)\n"+
				"    (as seq.empty %s)\n"+
				"    (seq.++ (%s (seq.extract xs 1 (- (seq.len xs) 1)))\n"+
				"            (seq.unit (seq.nth xs 0)))))",
			name, seqSort, seqSort, seqSort, prev,
		)
		decls = append(decls, body)
	}

	return decls
}

// generateTakeUnrolling generates bounded unrolling for _list_take.
//
// Level 0: (declare-fun _list_take_0 (Int (Seq Int)) (Seq Int))
// Level k: (define-fun _list_take_k ((n Int) (xs (Seq Int))) (Seq Int)
//
//	(ite (or (<= n 0) (= (seq.len xs) 0))
//	  (as seq.empty (Seq Int))
//	  (seq.++ (seq.unit (seq.nth xs 0))
//	          (_list_take_(k-1) (- n 1) (seq.extract xs 1 (- (seq.len xs) 1))))))
func generateTakeUnrolling(depth int, elemSort, seqSort string) []string {
	decls := make([]string, 0, depth+1)

	// Level 0: uninterpreted
	level0 := fmt.Sprintf("(declare-fun _list_take_0 (Int %s) %s)", seqSort, seqSort)
	decls = append(decls, level0)

	// Levels 1..depth
	for k := 1; k <= depth; k++ {
		name := fmt.Sprintf("_list_take_%d", k)
		prev := fmt.Sprintf("_list_take_%d", k-1)
		body := fmt.Sprintf(
			"(define-fun %s ((n Int) (xs %s)) %s\n"+
				"  (ite (or (<= n 0) (= (seq.len xs) 0))\n"+
				"    (as seq.empty %s)\n"+
				"    (seq.++ (seq.unit (seq.nth xs 0))\n"+
				"            (%s (- n 1) (seq.extract xs 1 (- (seq.len xs) 1))))))",
			name, seqSort, seqSort, seqSort, prev,
		)
		decls = append(decls, body)
	}

	return decls
}

// generateDropUnrolling generates bounded unrolling for _list_drop.
//
// Level 0: (declare-fun _list_drop_0 (Int (Seq Int)) (Seq Int))
// Level k: (define-fun _list_drop_k ((n Int) (xs (Seq Int))) (Seq Int)
//
//	(ite (or (<= n 0) (= (seq.len xs) 0))
//	  xs
//	  (_list_drop_(k-1) (- n 1) (seq.extract xs 1 (- (seq.len xs) 1)))))
func generateDropUnrolling(depth int, elemSort, seqSort string) []string {
	decls := make([]string, 0, depth+1)

	// Level 0: uninterpreted
	level0 := fmt.Sprintf("(declare-fun _list_drop_0 (Int %s) %s)", seqSort, seqSort)
	decls = append(decls, level0)

	// Levels 1..depth
	for k := 1; k <= depth; k++ {
		name := fmt.Sprintf("_list_drop_%d", k)
		prev := fmt.Sprintf("_list_drop_%d", k-1)
		body := fmt.Sprintf(
			"(define-fun %s ((n Int) (xs %s)) %s\n"+
				"  (ite (or (<= n 0) (= (seq.len xs) 0))\n"+
				"    xs\n"+
				"    (%s (- n 1) (seq.extract xs 1 (- (seq.len xs) 1)))))",
			name, seqSort, seqSort, prev,
		)
		decls = append(decls, body)
	}

	return decls
}
