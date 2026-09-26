package oggopus

// Ogg uses a CRC-32 that is unusual compared with the reflected CRC-32 in the
// standard library (hash/crc32). The parameters are (RFC 3533 section 6, and
// libogg framing.c):
//
//	polynomial: 0x04c11db7
//	initial value: 0x00000000
//	input reflected: no
//	output reflected: no
//	final XOR: 0x00000000
//
// Because there is no reflection, hash/crc32 (which only offers the reflected
// and Castagnoli forms) cannot compute it; the table and update loop below
// implement the MSB-first variant directly. The 32-bit checksum is then stored
// little-endian in the page header like every other Ogg integer field.
const crcPoly = 0x04c11db7

// crcTable is the 256-entry lookup table for the Ogg CRC-32. It is built once at
// package initialisation from crcPoly using the non-reflected construction.
var crcTable = makeCRCTable()

func makeCRCTable() [256]uint32 {
	var t [256]uint32
	for i := range t {
		r := uint32(i) << 24
		for range 8 {
			if r&0x80000000 != 0 {
				r = (r << 1) ^ crcPoly
			} else {
				r <<= 1
			}
		}
		t[i] = r
	}
	return t
}

// crc32 computes the Ogg CRC-32 of b starting from the given running value. Pass
// 0 for a fresh checksum. Callers that build a page in pieces can chain calls.
func crc32(crc uint32, b []byte) uint32 {
	for _, x := range b {
		crc = (crc << 8) ^ crcTable[byte(crc>>24)^x]
	}
	return crc
}
