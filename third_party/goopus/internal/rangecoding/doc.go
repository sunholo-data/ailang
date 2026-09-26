// Package rangecoding is a Go port of the libopus range coder (celt/entenc.c,
// entdec.c, entcode.c). The coder is a value-copyable struct over a shared
// []byte so snapshot is struct assignment and byte splicing is copy(), which the
// encoder's two RDO features depend on. It reproduces the libopus v1.6.1
// bit-level coder exactly and is held bit-exact by the refc differential gate;
// bit-exactness of the coder output is the sole correctness contract.
package rangecoding
