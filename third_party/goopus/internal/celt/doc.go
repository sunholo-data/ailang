// Package celt is a Go port of the libopus CELT codec (celt/): the decoder
// (phase 2) and the CELT-only encoder (phase 4). It carries per-file provenance
// references to the libopus v1.6.1 source (e.g. // celt/rate.c:249) and is held
// bit-exact against it by the refc differential gate in internal/reftest/oracle.
// Bit-exactness of the output is the sole correctness contract; the code is
// idiomatic Go where that does not obscure the mapping back to the C.
package celt
