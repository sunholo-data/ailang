package packet

// parseSize decodes one frame-length field from the front of data, using length
// as the number of logically available bytes (which can be smaller than
// len(data) when trailing padding has been reserved). It returns the number of
// header bytes consumed and the decoded frame size, or (-1, -1) on truncation.
//
// Length values below 252 occupy a single byte; larger values (up to 1275)
// occupy two bytes: size = 4*data[1] + data[0]. This mirrors parse_size() in
// libopus src/opus.c.
func parseSize(data []byte, length int) (nbytes, size int) {
	if length < 1 || len(data) < 1 {
		return -1, -1
	}
	if data[0] < 252 {
		return 1, int(data[0])
	}
	if length < 2 || len(data) < 2 {
		return -1, -1
	}
	return 2, 4*int(data[1]) + int(data[0])
}

// encodeSize writes one frame-length field to the front of data and returns the
// number of bytes written (1 or 2). It mirrors encode_size() in libopus
// src/opus.c and is the inverse of parseSize. The caller must ensure data has
// room for two bytes when size >= 252.
func encodeSize(size int, data []byte) int {
	if size < 252 {
		data[0] = byte(size)
		return 1
	}
	data[0] = byte(252 + (size & 0x3))
	data[1] = byte((size - int(data[0])) >> 2)
	return 2
}
