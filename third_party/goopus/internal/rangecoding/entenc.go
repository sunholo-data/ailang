/* Copyright (c) 2001-2011 Timothy B. Terriberry
   Copyright (c) 2008-2009 Xiph.Org Foundation */
/*
   Redistribution and use in source and binary forms, with or without
   modification, are permitted provided that the following conditions
   are met:

   - Redistributions of source code must retain the above copyright
   notice, this list of conditions and the following disclaimer.

   - Redistributions in binary form must reproduce the above copyright
   notice, this list of conditions and the following disclaimer in the
   documentation and/or other materials provided with the distribution.

   THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
   ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
   LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
   A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER
   OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
   EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
   PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
   PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
   LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
   NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
   SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// This file transliterates celt/entenc.c. It is a range encoder; see entdec.go
// and the references for implementation details.

package rangecoding

// Encoder is the entropy encoder context (celt/entcode.h struct ec_ctx, encoder
// view).
//
// This is a plain value struct whose only reference-typed field is buf, a shared
// []byte. Copying an Encoder (saved := *enc) copies the slice header, so the copy
// aliases the same backing array, exactly matching C's `enc_start_state = *enc`
// pointer semantics. That aliasing is what makes the encoder's two RDO features
// (two-pass coarse energy, theta RDO) expressible: snapshot is struct assignment,
// and byte splicing is copy() over the ec_get_buffer() / ec_range_bytes() window
// (docs/hard-parts.md 1). Do NOT hide buf behind an accessor that prevents that
// aliasing.
//
// A nil buf is the Go spelling of C's ALLOC_NONE sentinel (save_bytes==0).
type Encoder struct {
	// Buffered output.
	buf []byte
	// The size of the buffer.
	storage uint32
	// The offset at which the last byte containing raw bits was written.
	end_offs uint32
	// Bits that will be written at the end.
	end_window ecWindow
	// Number of valid bits in end_window.
	nend_bits int
	// The total number of whole bits written. This does not include partial bits
	// currently in the range coder.
	nbits_total int
	// The offset at which the next range coder byte will be written.
	offs uint32
	// The number of values in the current range.
	rng uint32
	// The low end of the current range.
	val uint32
	// The number of outstanding carry propagating symbols.
	ext uint32
	// A buffered output symbol, awaiting carry propagation.
	rem int
	// Nonzero if an error occurred.
	error int
}

func (_this *Encoder) write_byte(_value uint32) int {
	if _this.offs+_this.end_offs >= _this.storage {
		return -1
	}
	_this.buf[_this.offs] = byte(_value)
	_this.offs++
	return 0
}

func (_this *Encoder) write_byte_at_end(_value uint32) int {
	if _this.offs+_this.end_offs >= _this.storage {
		return -1
	}
	_this.end_offs++
	_this.buf[_this.storage-_this.end_offs] = byte(_value)
	return 0
}

// enc_carry_out outputs a symbol, with a carry bit.
// If there is a potential to propagate a carry over several symbols, they are
// buffered until it can be determined whether or not an actual carry will occur.
// If the counter for the buffered symbols overflows, then the stream becomes
// undecodable. This gives a theoretical limit of a few billion symbols in a
// single packet on 32-bit systems. The alternative is to truncate the range in
// order to force a carry, but requires similar carry tracking in the decoder,
// needlessly slowing it down.
func (_this *Encoder) enc_carry_out(_c int) {
	if _c != ecSymMax {
		/*No further carry propagation possible, flush buffer.*/
		carry := _c >> ecSymBits
		/*Don't output a byte on the first write.
		  This compare should be taken care of by branch-prediction thereafter.*/
		if _this.rem >= 0 {
			_this.error |= _this.write_byte(uint32(_this.rem + carry))
		}
		if _this.ext > 0 {
			sym := uint32(ecSymMax+carry) & ecSymMax
			for {
				_this.error |= _this.write_byte(sym)
				_this.ext--
				if _this.ext == 0 {
					break
				}
			}
		}
		_this.rem = _c & ecSymMax
	} else {
		_this.ext++
	}
}

func (_this *Encoder) enc_normalize() {
	/*If the range is too small, output some bits and rescale it.*/
	for _this.rng <= ecCodeBot {
		_this.enc_carry_out(int(_this.val >> ecCodeShift))
		/*Move the next-to-high-order symbol into the high-order position.*/
		_this.val = (_this.val << ecSymBits) & (ecCodeTop - 1)
		_this.rng <<= ecSymBits
		_this.nbits_total += ecSymBits
	}
}

// Init initializes the encoder over _buf (ec_enc_init); the buffer size is
// len(_buf).
func (_this *Encoder) Init(_buf []byte) {
	_this.buf = _buf
	_this.end_offs = 0
	_this.end_window = 0
	_this.nend_bits = 0
	/*This is the offset from which ec_tell() will subtract partial bits.*/
	_this.nbits_total = ecCodeBits + 1
	_this.offs = 0
	_this.rng = ecCodeTop
	_this.rem = -1
	_this.val = 0
	_this.ext = 0
	_this.storage = uint32(len(_buf))
	_this.error = 0
}

// Encode encodes a symbol given its frequency information (ec_encode).
// _fl: The cumulative frequency of all symbols that come before the one to be
// encoded.
// _fh: The cumulative frequency of all symbols up to and including the one to be
// encoded.
// _ft: The sum of the frequencies of all the symbols.
func (_this *Encoder) Encode(_fl, _fh, _ft uint32) {
	r := celtUdiv(_this.rng, _ft)
	if _fl > 0 {
		_this.val += _this.rng - r*(_ft-_fl)
		_this.rng = r * (_fh - _fl)
	} else {
		_this.rng -= r * (_ft - _fh)
	}
	_this.enc_normalize()
	_this.trace(traceEncode, _fl, _fh, _ft)
}

// EncodeBin is equivalent to Encode with _ft==1<<_bits (ec_encode_bin).
func (_this *Encoder) EncodeBin(_fl, _fh, _bits uint32) {
	r := _this.rng >> _bits
	if _fl > 0 {
		_this.val += _this.rng - r*((uint32(1)<<_bits)-_fl)
		_this.rng = r * (_fh - _fl)
	} else {
		_this.rng -= r * ((uint32(1) << _bits) - _fh)
	}
	_this.enc_normalize()
	_this.trace(traceEncodeBin, _fl, _fh, _bits)
}

// EncBitLogp encodes a bit that has a 1/(1<<_logp) probability of being a one
// (ec_enc_bit_logp).
func (_this *Encoder) EncBitLogp(_val int, _logp uint32) {
	var r uint32
	var s uint32
	var l uint32
	r = _this.rng
	l = _this.val
	s = r >> _logp
	r -= s
	if _val != 0 {
		_this.val = l + r
	}
	if _val != 0 {
		_this.rng = s
	} else {
		_this.rng = r
	}
	_this.enc_normalize()
	_this.trace(traceEncBitLogp, uint32(_val), _logp, 0)
}

// EncIcdf encodes a symbol given an "inverse" CDF table (ec_enc_icdf).
// _s:    The index of the symbol to encode.
// _icdf: The "inverse" CDF, monotonically non-increasing with last value 0.
// _ftb:  The number of bits of precision in the cumulative distribution.
func (_this *Encoder) EncIcdf(_s int, _icdf []byte, _ftb uint32) {
	r := _this.rng >> _ftb
	if _s > 0 {
		_this.val += _this.rng - r*uint32(_icdf[_s-1])
		_this.rng = r * (uint32(_icdf[_s-1]) - uint32(_icdf[_s]))
	} else {
		_this.rng -= r * uint32(_icdf[_s])
	}
	_this.enc_normalize()
	_this.trace(traceEncIcdf, uint32(_s), _ftb, 0)
}

// EncUint encodes a raw unsigned integer in the stream (ec_enc_uint).
// _fl: The integer to encode.
// _ft: The number of integers that can be encoded (one more than the max). This
// must be at least 2, and no more than 2**32-1.
func (_this *Encoder) EncUint(_fl, _ft uint32) {
	var ft uint32
	var fl uint32
	var ftb int
	/*In order to optimize EC_ILOG(), it is undefined for the value 0.*/
	/* celt_assert(_ft>1) */
	_ft--
	ftb = ecIlog(_ft)
	if ftb > ecUintBits {
		ftb -= ecUintBits
		ft = (_ft >> ftb) + 1
		fl = _fl >> ftb
		_this.Encode(fl, fl+1, ft)
		_this.EncBits(_fl&((uint32(1)<<ftb)-1), uint32(ftb))
	} else {
		_this.Encode(_fl, _fl+1, _ft+1)
	}
}

// EncBits encodes a sequence of raw bits in the stream (ec_enc_bits).
// _fl:   The bits to encode.
// _bits: The number of bits to encode. This must be between 1 and 25.
func (_this *Encoder) EncBits(_fl uint32, _bits uint32) {
	var window ecWindow
	var used int
	window = _this.end_window
	used = _this.nend_bits
	/* celt_assert(_bits>0) */
	if used+int(_bits) > ecWindowSize {
		for {
			_this.error |= _this.write_byte_at_end(window & ecSymMax)
			window >>= ecSymBits
			used -= ecSymBits
			if used < ecSymBits {
				break
			}
		}
	}
	window |= _fl << used
	used += int(_bits)
	_this.end_window = window
	_this.nend_bits = used
	_this.nbits_total += int(_bits)
	_this.trace(traceEncBits, _fl, _bits, 0)
}

// EncPatchInitialBits overwrites a few bits at the very start of an existing
// stream, after they have already been encoded (ec_enc_patch_initial_bits).
// _val:   The bits to encode (in the least _nbits significant bits).
// _nbits: The number of bits to overwrite. This must be no more than 8.
func (_this *Encoder) EncPatchInitialBits(_val, _nbits uint32) {
	var shift int
	var mask uint32
	/* celt_assert(_nbits<=EC_SYM_BITS) */
	shift = ecSymBits - int(_nbits)
	mask = ((1 << _nbits) - 1) << shift
	if _this.offs > 0 {
		/*The first byte has been finalized.*/
		_this.buf[0] = byte((uint32(_this.buf[0]) &^ mask) | _val<<shift)
	} else if _this.rem >= 0 {
		/*The first byte is still awaiting carry propagation.*/
		_this.rem = (_this.rem &^ int(mask)) | int(_val<<shift)
	} else if _this.rng <= (ecCodeTop >> _nbits) {
		/*The renormalization loop has never been run.*/
		_this.val = (_this.val &^ (mask << ecCodeShift)) |
			_val<<(ecCodeShift+shift)
	} else {
		/*The encoder hasn't even encoded _nbits of data yet.*/
		_this.error = -1
	}
}

// EncShrink compacts the data to fit in the target size (ec_enc_shrink).
// This moves up the raw bits at the end of the current buffer so they are at the
// end of the new buffer size.
// _size: The number of bytes in the new buffer.
func (_this *Encoder) EncShrink(_size uint32) {
	/* celt_assert(_this->offs+_this->end_offs<=_size) */
	copy(_this.buf[_size-_this.end_offs:_size],
		_this.buf[_this.storage-_this.end_offs:_this.storage])
	_this.storage = _size
}

// EncDone indicates that there are no more symbols to encode (ec_enc_done).
// All remaining output bytes are flushed to the output buffer. Init() must be
// called before the encoder can be used again.
func (_this *Encoder) EncDone() {
	var window ecWindow
	var used int
	var msk uint32
	var end uint32
	var l int
	/*We output the minimum number of bits that ensures that the symbols encoded
	  thus far will be decoded correctly regardless of the bits that follow.*/
	l = ecCodeBits - ecIlog(_this.rng)
	msk = (ecCodeTop - 1) >> l
	end = (_this.val + msk) &^ msk
	if (end | msk) >= _this.val+_this.rng {
		l++
		msk >>= 1
		end = (_this.val + msk) &^ msk
	}
	for l > 0 {
		_this.enc_carry_out(int(end >> ecCodeShift))
		end = (end << ecSymBits) & (ecCodeTop - 1)
		l -= ecSymBits
	}
	/*If we have a buffered byte flush it into the output buffer.*/
	if _this.rem >= 0 || _this.ext > 0 {
		_this.enc_carry_out(0)
	}
	/*If we have buffered extra bits, flush them as well.*/
	window = _this.end_window
	used = _this.nend_bits
	for used >= ecSymBits {
		_this.error |= _this.write_byte_at_end(window & ecSymMax)
		window >>= ecSymBits
		used -= ecSymBits
	}
	/*Clear any excess space and add any remaining extra bits to the last byte.*/
	if _this.error == 0 {
		if _this.buf != nil {
			clear(_this.buf[_this.offs : _this.storage-_this.end_offs])
		}
		if used > 0 {
			/*If there's no range coder data at all, give up.*/
			if _this.end_offs >= _this.storage {
				_this.error = -1
			} else {
				l = -l
				/*If we've busted, don't add too many extra bits to the last
				  byte; it would corrupt the range coder data, and that's more
				  important.*/
				if _this.offs+_this.end_offs >= _this.storage && l < used {
					window &= (1 << l) - 1
					_this.error = -1
				}
				_this.buf[_this.storage-_this.end_offs-1] |= byte(window)
			}
		}
	}
}

// Tell returns the number of whole bits used so far (ec_tell).
func (_this *Encoder) Tell() int { return ecTell(_this.nbits_total, _this.rng) }

// SetTellForSilence fast-forwards the whole-bit counter so that Tell() reports
// targetBits, mirroring celt_encoder.c:2006 (enc->nbits_total += tell -
// ec_tell(enc)) on the CELT silence path: with tell = nbCompressedBytes*8 the
// encoder pretends it has already filled every remaining bit with zeros, so every
// subsequent budget check fails and no further symbols are coded. This is the
// encoder-side twin of Decoder.SetTellForSilence (entdec.go:277) and, like it,
// is the only write accessor the CELT port needs on the bit counter.
func (_this *Encoder) SetTellForSilence(targetBits int) {
	_this.nbits_total += targetBits - _this.Tell()
}

// TellFrac returns the number of bits used so far scaled by 2**BITRES
// (ec_tell_frac).
func (_this *Encoder) TellFrac() uint32 { return ecTellFrac(_this.nbits_total, _this.rng) }

// Buffer returns the shared output buffer (ec_get_buffer). The returned slice
// aliases the coder's backing array so callers can splice bytes with copy(),
// which the RDO features depend on (docs/hard-parts.md 1).
func (_this *Encoder) Buffer() []byte { return _this.buf }

// RangeBytes returns the number of range coder bytes written so far, i.e., the
// length of the head window buf[0:offs] (ec_range_bytes).
func (_this *Encoder) RangeBytes() uint32 { return _this.offs }

// Storage returns the current size of the output buffer in bytes (the ec_ctx
// storage field, which EncShrink can reduce). CELT reads enc->storage*8 as a
// budget in quant_fine_energy (celt/quant_bands.c).
func (_this *Encoder) Storage() uint32 { return _this.storage }

// Error returns nonzero if an error has occurred (ec_get_error).
func (_this *Encoder) Error() int { return _this.error }

// Rng returns the current range register (the rng field), for tests and
// differential comparison.
func (_this *Encoder) Rng() uint32 { return _this.rng }

// Val returns the current low end of the range (the val field), for tests and
// differential comparison.
func (_this *Encoder) Val() uint32 { return _this.val }
