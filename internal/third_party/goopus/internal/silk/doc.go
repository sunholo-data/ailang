// Package silk is a Go port of the libopus SILK codec (silk/): the decoder
// (phase 3) and, optionally, the encoder (phase 5). It carries per-file
// provenance references to the libopus v1.6.1 source and is held bit-exact
// against it by the refc differential gate in internal/reftest/oracle;
// bit-exactness of the output is the sole correctness contract.
package silk
